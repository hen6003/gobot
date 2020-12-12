package main

import (
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hen6003/flago"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube"
)

var token string
var helpEmbed *discordgo.MessageEmbed
var dg *discordgo.Session

var messageNums map[string]int = make(map[string]int)

func main() {
	token = flago.NonFlags()[0]

	if token == "" {
		log.Println("No token provided. Please run: " + flago.ProgramName + " <bot token>")
		return
	}

	data, err := ioutil.ReadFile("msgsdata.save")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}
	dataArray := strings.Split(string(data), "\n")
	dataArray = dataArray[:len(dataArray)-1]

	for _, v := range dataArray {
		info := strings.Split(v, ":")

		num, err := strconv.Atoi(info[1])
		if err != nil {
			log.Panicf("failed reading data from file: %s", err)
		}

		messageNums[info[0]] = num
	}

	fields := make([]*discordgo.MessageEmbedField, 0)

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "!play [args]",
		Value: "search youtube for [args] and play the video",
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "!stop",
		Value: "stop playback in the server",
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "!img [args]",
		Value: "search yandex for [args] and sends it",
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "!msgs",
		Value: "sends how many messages you have sent",
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "!help",
		Value: "show this help",
	})

	helpEmbed = &discordgo.MessageEmbed{
		Description: "Help Menu",
	}

	helpEmbed.Fields = append(helpEmbed.Fields, fields...)

	// Create a new Discord session using the provided bot token.
	dg, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Println("Error creating Discord session: ", err)
		os.Exit(1)
	}

	dg.AddHandler(ready) // Register ready as a callback for the ready events.

	dg.AddHandler(messageCreate) // Register messageCreate as a callback for the messageCreate events.

	dg.AddHandler(guildCreate) // Register guildCreate as a callback for the guildCreate events.

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsGuildVoiceStates)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("Error opening Discord session: ", err)
		os.Exit(1)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println(flago.ProgramName + " is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

	log.Println("Writing data to save file")

	f, err := os.Create("msgsdata.save")
	if err != nil {
		log.Println(err)
		return
	}
	for i, v := range messageNums {
		vStr := strconv.Itoa(v)
		_, err := f.WriteString(i + ":" + vStr + "\n")
		if err != nil {
			log.Println(err)
			f.Close()
			return
		}
	}
	err = f.Close()
	if err != nil {
		log.Println(err)
		return
	}
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {

	// Set the playing status.
	s.UpdateStatus(0, "!help")
}

// from @Xeoncross on stackoverflow.com
func rankMap(values map[string]int) []string {
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range values {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	ranked := make([]string, len(values))
	for i, kv := range ss {
		ranked[i] = kv.Key
	}
	return ranked
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if _, ok := messageNums[m.Author.ID]; !ok {
		messageNums[m.Author.ID] = 0
	}
	messageNums[m.Author.ID]++

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

	msg := strings.Split(m.Content, " ")

	var msgStr string
	for _, v := range msg[1:] {
		msgStr += v + " "
	}

	switch msg[0] {
	case "!play":
		s.ChannelMessageSend(c.ID, "Playing")

		videoID := search(msgStr)

		s.ChannelMessageSend(c.ID, "Found: https://youtube.com/watch?v="+videoID)

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID, videoID)
				if err != nil {
					log.Println("Error playing sound:", err)
				}

				return
			}
		}

		s.ChannelMessageSend(c.ID, "ERROR: You are not in any voice channels")

	case "!stop":
		s.ChannelMessageSend(c.ID, "Stopping, Cya")
		for _, vcs := range s.VoiceConnections {
			if vcs.GuildID == g.ID {
				vcs.Disconnect()
			}
		}

	case "!img":
		imgUrl := imgSearch(msgStr)

		s.ChannelMessageDelete(c.ID, m.ID)

		imgEmbed := &discordgo.MessageEmbed{
			Description: "Search Term: " + msgStr,
			Image:       &discordgo.MessageEmbedImage{URL: imgUrl},
		}

		imgEmbed.Color = embedColourGen()

		s.ChannelMessageSendEmbed(c.ID, imgEmbed)

	case "!msgs":
		msgsEmbed := &discordgo.MessageEmbed{}

		if len(msg) > 1 {
			if msg[1] == "leaderboard" {
				ranks := rankMap(messageNums)

				fields := make([]*discordgo.MessageEmbedField, 0)

				for _, v := range ranks {
					msgSent := strconv.Itoa(messageNums[v])

					user, err := dg.User(v)
					if err != nil {
						log.Println("Error finding user:", err)
					}

					username := user.Username

					fields = append(fields, &discordgo.MessageEmbedField{
						Name:  "User: " + username,
						Value: "Messages sent: " + msgSent,
					})

					msgsEmbed.Fields = fields
				}
			} else if len(m.Mentions) > 0 {
				fields := makeMsgEmbed(m.Mentions[0].Username, strconv.Itoa(messageNums[m.Mentions[0].ID]))
				msgsEmbed.Fields = append(msgsEmbed.Fields, fields...)
			}
		} else {
			fields := makeMsgEmbed(m.Author.Username, strconv.Itoa(messageNums[m.Author.ID]))

			msgsEmbed.Fields = append(msgsEmbed.Fields, fields...)
		}

		msgsEmbed.Color = embedColourGen()

		s.ChannelMessageSendEmbed(c.ID, msgsEmbed)

	case "!help":
		helpEmbed.Color = embedColourGen()

		s.ChannelMessageSendEmbed(c.ID, helpEmbed)
	}
}

func makeMsgEmbed(username string, msgSent string) []*discordgo.MessageEmbedField {
	fields := make([]*discordgo.MessageEmbedField, 0)

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  "User: " + username,
		Value: "Messages sent: " + msgSent,
	})

	return fields
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

func embedColourGen() int {
	rand.Seed(time.Now().UnixNano())
	options := []int{0, 1752220, 3066993, 3447003, 10181046, 15844367, 15105570, 15158332, 9807270, 8359053, 3426654, 1146986, 2067276, 2123412, 7419530, 12745742, 11027200, 10038562, 9936031, 12370112, 2899536, 16580705, 12320855}

	return options[rand.Intn(22)]
}
