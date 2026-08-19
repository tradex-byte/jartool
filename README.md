# JAR Tool

Command-line tool for decompiling, text replacement, and compiling JAR files.

## Compilation

### Windows
go build -o jartool.exe jartool.go

### Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o jartool jartool.go

## Usage

Decompile JAR file:
jartool -d input.jar -o output_dir

Replace text in files:
jartool -r output_dir -s search_string -n replace_string

Compile JAR file:
jartool -c input_dir -o output.jar

Display help:
jartool -h

## Examples

Decompile an application:
jartool -d app.jar -o decompiled

Replace URL in decompiled files:
jartool -r decompiled -s http://127.0.0.1 -n https://example.com

Compile modified files:
jartool -c decompiled -o app_modified.jar

## Parameters

-d JAR file to decompile
-c Directory to compile
-o Output file or directory
-r Directory for text replacement
-s String to search for
-n String to replace with
-h Show help screen

## Features

Automatic Java download if not present
Java stored in java directory next to executable
Cross-platform (Windows and Linux)
No administrator privileges required
Works on older operating systems
Single binary file

## Directory Structure

After first run, the following structure is created:

jartool.exe (or jartool)
java/
  jdk8u502-b07/
    bin/
      jar.exe (or jar)
    lib/
    ...

## Requirements

Go 1.10 or higher (for compilation only)
Operating System: Windows or Linux
Internet connection (for automatic Java download)
No external dependencies for the compiled binary

## Notes

The tool downloads OpenJDK 8 automatically
All operations are performed in the current working directory
Text replacement works recursively on all files
Existing JAR files are overwritten during compilation
Java is only downloaded once and cached locally
