# Overview
This is a terminal text editor developed entirely with ChatGPT for testing purposes to determine how well a software developer can use AI for software development.

# Development

Start program.
```sh
go run .
```

## Prompt
In order for ChatGPT to adapt to the existing files, the contents of the relevant files in each prompt are required.
How can this be achieved? Here are two options. Ubuntu 24.04 with Wayland is required.

### One-liner version
Copy the command sequence and paste it into your console in the project folder.
```sh
{ echo "Give me back only the part of the code that should be adjusted."; echo; find . -type f -name "*.go" -not -path "./vendor/*" -not -path "./testdata/*" -exec echo "===== {} =====" \; -exec cat {} \; ; } | wl-copy
```

### Script version
Run the following script in the project folder in the console to copy all go files to your clipboard.
```sh
./copy-go-files-to-clipboard.sh
```
