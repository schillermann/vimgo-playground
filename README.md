# Overview
This is a terminal-based text editor developed entirely with ChatGPT model GPT-5 for testing purposes to determine with which mindset a software developer can use AI for software development.
I'm using blog post [Build Your Own Text Editor](http://viewsourcecode.org/snaptoken/kilo/index.html) with kilo as a guide.

# Development

Start program.
```sh
go run .
```

# Keys
|Keys    |Description|
|--------|-----------|
|`Ctrl+Q`|Quit       |


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

# Experiences

## Special Case

ChatGPT can make a common suggestion that might not work for your specific case.

**Example:**
The key press input runs synchronously and thus blocks the entire editor process.
Therefore, the key press input should be asynchronous.
ChatGPT suggests moving the key press input to a separate goroutine.
Separating the key press input from the main process so it's no longer blocked seems logical at first glance.
What happens after I start the editor with the code change?
It no longer accepts key press. Why is that?
The key press input query from `os.Stdin` remains synchronous, even if it's moved to an asynchronous goroutine.
As long as there's no key press, the goroutine won't proceed.
The select statement waits a maximum of 100ms for a return from the goroutine.
If no key press is received, the goroutine is no longer waited for and is therefore permanently blocked, also known as a goroutine leak.
Next, another goroutine is started, which is also blocked after 100ms if it doesn't receive a key press.
And so it continues. Now we have many blocked goroutines, all connected to `os.Stdin`.
Which goroutine receives which byte key press is random.
The goroutines can no longer pass the key press to the main process, because they are blocked.
As a result, no key press input reaches the main program.
I described the problem to ChatGPT.
It then suggested that many goroutines were being started and blocked after 100ms.
It suggested, instead of spawning a new goroutine every loop iteration, use one long-lived goroutine whose only job is reading from `os.Stdin` and sending key presses to a channel.
This solved my problem, and the main process can process key presses again.

The ChatGPT hadn't detected the problem beforehand.
He would have had the opportunity to examine my code in a single file of 275 lines.
In my case, the error was immediately apparent, in more complex programs, it can lead to data loss before the error is detected.
Therefore, it remains the developer's responsibility to have sufficient experience to recognize such errors in the code beforehand, even before it is committed to Git.

## Alternatives 

ChatGPT suggests common approaches, but not specific ones.

**Example:**
Key press input blocks the main process.
ChatGPT should provide me with solutions to unblock the key press input.
It first suggests the simplest solution using `time.Ticker`.
In this solution the `time.Ticker` is set to 2 seconds, and `select` receives a signal when no key has been pressed, allowing another process to be handled, otherwise it reacts to the key press.
Then I asked for another solution.
It suggested moving the key press input to a goroutine.
Even with the information that I wanted to use the editor for Linux, it didn't point out that I could also use `VMIN` and `VTIME` to achieve non-blocking or timeout-based reads of the key press input directly via Termios.
Only after I explicitly mentioned `VMIN` and `VTIME` did it offer a solution for this as well.

I only became aware of this possibility by reading the Termios documentation.
ChatGPT doesn't save you from having to read the documentation.

## Verbose Code

Simple logic is unnecessarily inflated.
Sure, the code change might work, but who ultimately has to solve the tricky problems, when the AI gets stuck?
The developer, of course.
And what does a developer need to solve a problem?
A good code overview and that's only possible if the code is properly structured.

**Example:**
Using a script, I copied the contents of the code files to the clipboard as a prompt so that ChatGPT could provide the context for each task.
However, this meant that after a while, my chat window became very cluttered.
The Chrome browser was already starting to complain. So, I needed an alternative. Therefore, I started working with the Codex CLI in the terminal.
I gave him a simple initial task, the cursor should not move off the screen at the bottom or right.
As a solution, I received approximately 15 lines of code in which he created a function containing the logic to check if the cursor is outside the off-screen area, if so, it should reset the cursor.
This seemed too bloated to me, as it would have been much simpler to use two `if` statements to increment the cursor, and the cursor wouldn't have needed to be reset.
I gave him a second chance. The second time, he figured out how to add an `if` statement around each cursor increment.
So it's still important to understand the code changes and above all to have your own solution ready, at least a rough one, which you can compare with the suggested solution, because otherwise, the maintainability of the code is quickly compromised.
