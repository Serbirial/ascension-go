package fs

import (
	"log"
	"os"
	"os/exec"
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

	// Set up dca command, reading from ffmpegOut and writing to file
	dca := exec.Command("/home/summers/dca")
	dca.Stdin = ffmpegOut
	dca.Stdout = outFile

	// Start ffmpeg first
	if err := ffmpeg.Start(); err != nil {
		log.Fatalf("[Converter] Failed to start ffmpeg: %v", err)
	}

	// Then start dca (which reads from ffmpeg)
	if err := dca.Start(); err != nil {
		log.Fatalf("[Converter] Failed to start dca: %v", err)
	}

	// Wait for both to finish
	if err := ffmpeg.Wait(); err != nil {
		log.Fatalf("[Converter] ffmpeg exited with error: %v", err)
	}
	if err := dca.Wait(); err != nil {
		log.Fatalf("[Converter] dca exited with error: %v", err)
	}

	log.Printf("[Converter] Successfully wrote DCA file to: %s\n", outputFilePath)
	// Remove the original file
	err = os.Remove(file)
	if err != nil {
		log.Fatalf("[Converter] failed to remove original file: %v", err)

	}
	return outputFilePath
}
