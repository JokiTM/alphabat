package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// usageError marks errors caused by invalid user input.
// main prints the usage text for them and exits with code 2.
type usageError string

func (e usageError) Error() string { return string(e) }

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, `Usage: %s [command] [argument]

Commands:
  (none)             print battery level in percent
  sleep              print current sleep timer
  sleep <minutes>    set sleep timer (0-255, e.g. 10, 20, 30)
  voice              print whether voice prompts are enabled
  voice on|off       enable or disable voice prompts
`, os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if err := run(flag.Args()); err != nil {
		var ue usageError
		if errors.As(err, &ue) {
			fmt.Fprintf(os.Stderr, "%v\n\n", err)
			flag.Usage()
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// run executes the given command. Returning instead of calling os.Exit
// ensures that deferred cleanup (closing the device) always runs.
func run(args []string) error {
	h, err := Open()
	if err != nil {
		return fmt.Errorf("initializing device: %w", err)
	}
	defer h.Close()

	if len(args) == 0 {
		level, err := h.Query(BatteryLevel)
		if err != nil {
			return err
		}
		fmt.Println(level)
		return nil
	}

	hasArg := len(args) > 1

	switch args[0] {
	case "sleep":
		if !hasArg {
			state, err := h.Query(SleepState)
			if err != nil {
				return err
			}
			fmt.Println(state)
			return nil
		}
		min, err := strconv.ParseUint(args[1], 10, 8) // base 10, max. 8 bit
		if err != nil {
			return usageError(fmt.Sprintf("Invalid sleep time %q", args[1]))
		}
		if _, err := h.Set(0x21bb1200 | Command(min)); err != nil { // 20 → 0x14
			return err
		}
	case "voice":
		if !hasArg {
			state, err := h.Query(VoiceState)
			if err != nil {
				return err
			}
			switch state {
			case 1:
				fmt.Println("voice prompts enabled")
			case 0:
				fmt.Println("voice prompts disabled")
			default:
				fmt.Println("unknown voice prompt state:", state)
			}
			return nil
		}
		var cmd Command
		switch args[1] {
		case "on":
			cmd = VoicePromptsOn
		case "off":
			cmd = VoicePromptsOff
		default:
			return usageError(fmt.Sprintf("Invalid voice option %q, expected \"on\" or \"off\"", args[1]))
		}
		if _, err := h.Set(cmd); err != nil {
			return err
		}
	default:
		return usageError(fmt.Sprintf("Invalid command %q", args[0]))
	}

	// Give the headset time to apply the setting before the device is closed.
	time.Sleep(timeout)
	return nil
}
