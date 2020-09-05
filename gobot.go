package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

var token string

func main() {

	if token == "" {
		fmt.Println("No token provided. Please run: airhorn -t <bot token>")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "!help")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Find the channel that the message came from.
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Could not find channel.
		return
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		// Could not find guild.
		return
	}

	if strings.HasPrefix(m.Content, "!play") {
		s.ChannelMessageSend(c.ID, "Playing")

		msg := strings.Split(m.Content, " ")

		var msgStr string
		for _, v := range msg[1:] {
			msgStr += v
		}

		videoID := search(msgStr)

		s.ChannelMessageSend(c.ID, "Found: https://youtube.com/watch?v="+videoID)

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID, videoID)
				if err != nil {
					fmt.Println("Error playing sound:", err)
				}

				return
			}
		}
	}

	if strings.HasPrefix(m.Content, "!playid") {
		s.ChannelMessageSend(c.ID, "Playing")

		msg := strings.Split(m.Content, " ")

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID, msg[1])
				if err != nil {
					fmt.Println("Error playing sound:", err)
				}

				return
			}
		}
	}

	if strings.HasPrefix(m.Content, "!stop") {
		s.ChannelMessageSend(c.ID, "Stopping, Cya")
		for _, vcs := range s.VoiceConnections {
			if vcs.GuildID == g.ID {
				vcs.Disconnect()
			}
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {

	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessageSend(channel.ID, "owo")
			return
		}
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID, channelID string, videoID string) (err error) {
	client := youtube.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		panic(err)
	}

	resp, err := client.GetStream(video, &video.Formats[0])
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Encoding a file and saving it to disk
	file, err := dca.EncodeMem(resp.Body, dca.StdEncodeOptions)
	if err != nil {
		panic(err)
	}

	// Make sure everything is cleaned up, that for example the encoding process if any issues happened isnt lingering around
	defer file.Cleanup()

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Start speaking.
	vc.Speaking(true)

	// Source is an OpusReader, both EncodeSession and decoder implements opusreader
	done := make(chan error)
	dca.NewStream(file, vc, done)
	err = <-done
	if err != nil && err != io.EOF {
		// Handle the error
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}
