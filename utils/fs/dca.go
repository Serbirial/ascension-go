package fs

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"os/exec"
	"sync/atomic"
)

func convertToDCA(file string) string {
	log.Printf("[Converter] Converting %s to discord accepted DCA", file)
	var outputFilePath string = file + ".dca"
	// Create output file
	outFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("[Converter] Failed to create output file: %v", err)
	}
	defer outFile.Close()

	ffmpeg := exec.Command("ffmpeg", "-i", file, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")

	// Get ffmpeg's stdout (raw PCM stream)
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Fatalf("[Converter] Failed to get ffmpeg stdout: %v", err)
	}
	// Also get any errors
	ffmpegErr, err := ffmpeg.StderrPipe()
	if err != nil {
		log.Fatalf("[Converter] Failed to get ffmpeg stderr: %v", err)
	}

	// Set up dca command, reading from ffmpegOut and writing to file
	dca := exec.Command("/home/summers/dca", "-ab", "128") // ab is bitrate
	dca.Stdin = ffmpegOut
	dca.Stdout = outFile

	//ffmpeg.Stderr = os.Stderr // Show logs in terminal

	// Start ffmpeg first
	if err := ffmpeg.Start(); err != nil {
		log.Fatalf("[Converter] Failed to start ffmpeg: %v", err)
	}

	// Read stderr in background
	var ffmpegStderr []byte
	done := make(chan struct{})
	go func() {
		ffmpegStderr, _ = io.ReadAll(ffmpegErr)
		close(done)
	}()

	// Then start dca (which reads from ffmpeg)
	if err := dca.Start(); err != nil {
		log.Fatalf("[Converter] Failed to start dca: %v", err)
	}

	// Wait for both to finish
	if err := ffmpeg.Wait(); err != nil {
		log.Printf("[Converter] ffmpeg stderr:\n%s", string(ffmpegStderr))

		err = os.Remove(file)
		if err != nil {
			log.Fatalf("[Converter] failed to remove original file: %v", err)

		}
		err = os.Remove(outFile.Name())
		if err != nil {
			log.Fatalf("[Converter] failed to remove original file: %v", err)

		}
		log.Fatalf("[Converter] ffmpeg exited with error: %v", err) //, string(ffmpegErrOutput))

	}
	if err := dca.Wait(); err != nil {
		log.Fatalf("[Converter] dca exited with error: %v", err)
		err = os.Remove(file)
		if err != nil {
			log.Fatalf("[Converter] failed to remove original file: %v", err)

		}
		err = os.Remove(outFile.Name())
		if err != nil {
			log.Fatalf("[Converter] failed to remove original file: %v", err)

		}
	}

	log.Printf("[Converter] Successfully wrote DCA file to: %s\n", outputFilePath)
	// Remove the original file
	err = os.Remove(file)
	if err != nil {
		log.Fatalf("[Converter] failed to remove original file: %v", err)

	}
	return outputFilePath
}

func ConvertToDCALive(in chan []byte, out chan []byte, stop chan bool) {
	// Set up dca command, reading from ffmpegOut and writing to file
	dca := exec.Command("/home/summers/dca", "-ab", "128") // ab is bitrate
	defer dca.Process.Kill()

	pipeOut, pipeIn, err := os.Pipe()
	if err != nil {
		log.Fatalf("[Converter] Failed to make DCA live pipe: %v", err)

	}
	dca.Stdin = pipeIn
	dca.Stdout = pipeOut

	defer pipeOut.Close()
	defer pipeIn.Close()

	var shouldStop int32 = 0 // 1 = true, 0 = false

	// Write to dca.Stdin (PCM data)
	go func() {
		for {
			if atomic.LoadInt32(&shouldStop) == 1 {
				return
			}
			data, ok := <-in
			if !ok {
				log.Println("DCA Converter Channel closed")
				return
			}
			_, err := pipeIn.Write(data)
			if err != nil {
				log.Println("DCA Converter had error during pipe write: " + err.Error())
				return
			}
		}
	}()

	// Read from dca.Stdout and send processed DCA data to out
	go func() {
		var opuslen int16

		for {
			if atomic.LoadInt32(&shouldStop) == 1 {
				return
			}
			err := binary.Read(pipeOut, binary.LittleEndian, &opuslen)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				log.Println("[Converter] Error reading frame length:", err)
				break
			}
			data := make([]byte, opuslen)
			err = binary.Read(pipeOut, binary.LittleEndian, &data)
			if err != nil {
				log.Println("[Converter] Error reading frame data:", err)
				break
			}
			out <- data
		}
	}()

	// Start dca
	if err := dca.Start(); err != nil {
		log.Fatalf("[Converter] Failed to start dca: %v", err)
	}

	select {
	case <-stop:
		atomic.StoreInt32(&shouldStop, 1)
		dca.Process.Kill()
		log.Printf("[Converter] Done converting DCA Live")
		return
	}

}
