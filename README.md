# Audio Chunker

A CLI tool for removing the audio from silence from an audio file.

## Requirements

- **[FFmpeg](https://ffmpeg.org/)** must be installed and available in your system's `PATH`.
  - This tool relies on FFmpeg for media processing.
  - You can check if FFmpeg is installed by running:
    ```sh
    ffmpeg -version
    ```
  - If you see version information, FFmpeg is installed.

## Installation

Clone the repository and build the binary:

```sh
git clone https://github.com/cmdl-dev/audio-chunker.git
cd audio-chunker
go build -o audiochunker
```

Or run directly with Go:

```sh
go run main.go [flags]
```

## Usage

```sh
audiochunker -f <input_file> -o <output_dir> [options]
```

### Required Flags

- `-f, --file string`  
  Path to the input file to be processed.

- `-o, --out string`  
  Directory where the output will be saved.

### Optional Flags

- `-n, --noise int`  
  Maximum noise limit to apply during processing.  
  Default: `0`

- `-t, --threshold float`  
  Threshold value for processing (e.g., 0.5).  
  Default: `0.0`

- `-h, --help`  
  Show help message.

## Example

```sh
audiochunker -f input.txt -o ./output -n 10 -t 0.5
```

## Help

To see all available options:

```sh
audiochunker -h
```
