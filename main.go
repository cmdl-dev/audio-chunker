package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type Silence struct {
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Duration float64 `json:"duration"`
}

type Chunk struct {
	start    float64
	end      float64
	duration float64
}

func detectSilences(file string, noise int, threshold float64) ([]Silence, error) {
	// FFmpeg command to detect silence
	af := fmt.Sprintf("silencedetect=noise=%ddB:d=%.2f", noise, threshold)

	cmd := exec.Command(
		"ffmpeg",
		"-i", file,
		"-af", af,
		"-f", "null", "-",
	)

	// FFmpeg prints silence info to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Error getting stderr:", err)
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting ffmpeg:", err)
		return nil, err
	}

	// Regex to match silence_start and silence_end
	silenceStartRe := regexp.MustCompile(`silence_start: ([\d\.]+)`)
	silenceEndRe := regexp.MustCompile(`silence_end: ([\d\.]+) \| silence_duration: ([\d\.]+)`)

	var silences []Silence
	var currentStart *float64

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if silenceStartRe.MatchString(line) {
			matches := silenceStartRe.FindStringSubmatch(line)
			start, _ := strconv.ParseFloat(matches[1], 64)
			currentStart = &start
		}
		if silenceEndRe.MatchString(line) && currentStart != nil {
			matches := silenceEndRe.FindStringSubmatch(line)
			end, _ := strconv.ParseFloat(matches[1], 64)
			duration, _ := strconv.ParseFloat(matches[2], 64)
			silences = append(silences, Silence{
				Start:    *currentStart,
				End:      end,
				Duration: duration,
			})
			currentStart = nil
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading stderr:", err)
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println("FFmpeg finished with error:", err)
	}
	return silences, nil
}

func chunkAudioSegment(inputFile, outputDir string, start, end float64) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	ext := filepath.Ext(inputFile)

	// Output pattern: outputDir/chunk_%03d.mp3
	outputPattern := filepath.Join(outputDir, fmt.Sprintf("chunk_%d-%d%s", int(start), int(end), ext))

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%.2f", start),
		"-to", fmt.Sprintf("%.2f", end),
		"-i", inputFile,
		"-c", "copy",
		outputPattern,
	)

	// Optional: print ffmpeg output for debugging
	// cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputPattern, nil
}
func compressAudio(inputFile, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	fileName := strings.Replace(filepath.Base(inputFile), filepath.Ext(inputFile), "", 1)
	out := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", fileName))

	cmd := exec.Command(
		"ffmpeg",
		"-i", inputFile,
		out,
	)

	// Optional: print ffmpeg output for debugging
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {

	var (
		file      string
		outDir    string
		minDur    int
		noise     int
		threshold float64
	)

	var rootCmd = &cobra.Command{
		Use:   "audiochunker",
		Short: "A CLI tool for removing the silence from an audio file",
		Long:  `A CLI tool for removing the silence from an audio file`,
		Run: func(cmd *cobra.Command, args []string) {
			// Check required flags
			if file == "" || outDir == "" {
				fmt.Fprintln(os.Stderr, "Error: -f and -o are required")
				cmd.Usage()
				os.Exit(1)
			}

			removeSilence(file, outDir, noise, minDur, threshold)
		},
	}

	// Define flags with detailed descriptions
	rootCmd.Flags().StringVarP(
		&file,
		"file",
		"f",
		"",
		"Path to the input file to be processed (required)",
	)
	rootCmd.Flags().StringVarP(
		&outDir,
		"out",
		"o",
		"",
		"Directory where the output will be saved (required)",
	)
	rootCmd.Flags().IntVarP(
		&minDur,
		"minDir",
		"m",
		3,
		"Minimum duration in seconds that the clip has to be (optional)",
	)
	rootCmd.Flags().Float64VarP(
		&threshold,
		"threshold",
		"t",
		0.5,
		"Threshold for Silence duration (e.g., 0.5) (optional)",
	)
	rootCmd.Flags().IntVarP(
		&noise,
		"noise",
		"n",
		-30,
		"Maximum noise limit to apply during processing (optional)",
	)

	// Mark required flags
	rootCmd.MarkFlagRequired("file")
	rootCmd.MarkFlagRequired("out")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func removeSilence(file, outDir string, noiseLimit, minDur int, threshold float64) {
	res, err := detectSilences(file, noiseLimit, threshold)
	if err != nil {
		log.Fatal(err)
	}

	var chunks []Chunk
	var cursor float64 = 0
	for _, r := range res {
		newChunk := Chunk{
			start:    cursor,
			end:      r.Start,
			duration: r.Start - cursor,
		}
		cursor = r.End

		if float64(newChunk.duration) > float64(minDur) {
			chunks = append(chunks, newChunk)
		}
	}

	fmt.Println("Chunking Audio Segments")
	for i, r := range chunks {
		fmt.Printf("\r%d/%d", i, len(chunks))
		if r.duration < float64(minDur) {
			continue
		}
		_, err := chunkAudioSegment(file, outDir, r.start, r.end)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("\nSaved to %s\n", outDir)
}
