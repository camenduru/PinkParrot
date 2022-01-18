package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/fsnotify/fsnotify"
	"github.com/gempir/go-twitch-irc"
	"github.com/getlantern/systray"
	"mvdan.cc/xurls/v2"

	icon "./icons"
)

var (
	settings map[string]string
)

func onReady() {

	getFiles()

	go watch()

	systray.SetIcon(icon.Data)
	systray.SetTitle("PinkParot v1.2 (dev:camenduru)")
	systray.SetTooltip("PinkParot v1.2 (dev:camenduru)")

	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(exe)
	mSettings := systray.AddMenuItem("Settings", "settings.txt")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit")

	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				exec.Command(`notepad`, exePath+`\settings.txt`).Run()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

	if strings.Compare(settings["oauth"], "") != 0 {
		go task()
	}
}

func GoogleSpeak(text, language string) error {
	url := fmt.Sprintf("http://translate.google.com/translate_tts?ie=UTF-8&total=1&idx=0&textlen=32&client=tw-ob&q=%s&tl=%s", url.QueryEscape(text), language)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer response.Body.Close()

	streamer, format, err := mp3.Decode(response.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done

	return nil
}

func onExit() {
}

func main() {
	systray.Run(onReady, onExit)
}

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					systray.Quit()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add("settings.txt")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func getFiles() {
	// settings.txt
	settings = make(map[string]string)
	settingsTxt, errSettings := os.Open("settings.txt")
	if errSettings != nil {
		log.Fatal(errSettings)
	}
	defer settingsTxt.Close()
	settingsScanner := bufio.NewScanner(settingsTxt)
	for settingsScanner.Scan() {
		settingKeyValue := strings.Split(string(settingsScanner.Text()), "~")
		settings[strings.TrimSpace(settingKeyValue[0])] = strings.TrimSpace(settingKeyValue[1])
	}
	if errSettingsScanner := settingsScanner.Err(); errSettingsScanner != nil {
		log.Fatal(errSettingsScanner)
	}
}

// Twitch
func task() {

	// Twitch
	client := twitch.NewClient(settings["username"], settings["oauth"])

	// Get Twitch Chat
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		to := settings["to"]
		text := message.Message

		// Check URL
		rxRelaxed := xurls.Relaxed()
		url := rxRelaxed.FindString(text)
		if url != "" {
			text = strings.ReplaceAll(text, url, "url")
		}

		if settings["auto"] == "on" {
			from := "auto"
			src, translated, err := TranslateWithParams(text, TranslationParams{From: from, To: to})
			if err != nil {
				panic(err)
			}
			if src == to {
				to = "en"
				src, translated, err := TranslateWithParams(text, TranslationParams{From: from, To: to})
				if err != nil {
					panic(err)
				}
				if settings["translate"] == "on" {
					client.Say(settings["channel"], fmt.Sprintf("/me %s [by %s] | (%s > %s) \n", translated, message.User.Name, src, to))
				}
				if settings["audio"] == "on" {
					if settings["read_username"] == "on" {
						GoogleSpeak(message.User.Name, "en")
					}
					GoogleSpeak(translated, to)
				}
			} else {
				if settings["translate"] == "on" {
					client.Say(settings["channel"], fmt.Sprintf("/me %s [by %s] | (%s > %s) \n", translated, message.User.Name, src, to))
				}
				if settings["audio"] == "on" {
					if settings["read_username"] == "on" {
						GoogleSpeak(message.User.Name, "en")
					}
					GoogleSpeak(translated, to)
				}
			}

		} else {

			if strings.HasPrefix(text, settings["first_char"]) || strings.HasPrefix(text, settings["jp_first_char"]) {
				text = strings.Replace(text, settings["first_char"], "", -1)
				text = strings.Replace(text, settings["jp_first_char"], "", -1)
				from := "auto"
				src, translated, err := TranslateWithParams(text, TranslationParams{From: from, To: to})
				if err != nil {
					panic(err)
				}
				if src == to {
					to = "en"
					src, translated, err := TranslateWithParams(text, TranslationParams{From: from, To: to})
					if err != nil {
						panic(err)
					}
					if settings["translate"] == "on" {
						client.Say(settings["channel"], fmt.Sprintf("/me %s [by %s] | (%s > %s) \n", translated, message.User.Name, src, to))
					}
					if settings["audio"] == "on" {
						if settings["read_username"] == "on" {
							GoogleSpeak(message.User.Name, "en")
						}
						GoogleSpeak(translated, to)
					}
				} else {
					if settings["translate"] == "on" {
						client.Say(settings["channel"], fmt.Sprintf("/me %s [by %s] | (%s > %s) \n", translated, message.User.Name, src, to))
					}
					if settings["audio"] == "on" {
						if settings["read_username"] == "on" {
							GoogleSpeak(message.User.Name, "en")
						}
						GoogleSpeak(translated, to)
					}
				}

			}
		}
	})

	// Twitch Join Channel
	client.Join(settings["channel"])
	errclient := client.Connect()
	if errclient != nil {
		panic(errclient)
	}

}
