package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"syscall"
	"flag"
	"strings"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

var token string

func main() {
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error while creating a Discord session,", err)
		return
	}

	discord.AddHandler(newMessage)

	err = discord.Open()
	if err != nil {
		fmt.Println("Error while opening connection,", err)
	}
	defer discord.Close()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

type Anime struct {
	Data []AnimeInfos `json:"data"`
}

type AnimeInfos struct {
	ID         string `json:"id"`
	Attributes struct {
		Synopsis string `json:"synopsis"`
		Titles   struct {
			En   string `json:"en"`
			EnJp string `json:"en_jp"`
			JaJp string `json:"ja_jp"`
		} `json:"titles"`
		Subtype     string `json:"subtype"`
		Status      string `json:"status"`
		PosterImage struct {
			Original string `json:"original"`
		} `json:"posterImage"`
		EpisodeCount int    `json:"episodeCount"`
		ShowType     string `json:"showType"`
	} `json:"attributes"`
}

/**
	Callback called whenever a new message is posted on a channel
 */
func newMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	formattedMessage := strings.Join(strings.Fields(message.Content), "%20")

	if strings.HasPrefix(formattedMessage, "_anime") {

		name := strings.TrimPrefix(formattedMessage, "_anime")

		response, err := http.Get("https://kitsu.io/api/edge/anime?filter[text]={" + name + "}&page[limit]=1") //TODO Sort by popular
		if err != nil || response.Status != "200 OK" {
			fmt.Println("Request failed (Status : "+ response.Status + "). Cannot get anime.\n", err)
			return
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error while reading request body,", err)
			return
		}

		var anime Anime
		parseAnimeInfos(body, &anime)

		if len(anime.Data) < 0 {
			fmt.Println("Didn't find any related anime with given name")
		} else {
			createAndSendEmbedMessage(session, message.ChannelID, anime.Data[0])
		}
	}
}

func parseAnimeInfos(body []byte, anime *Anime) {
	json.Unmarshal([]byte(body), &anime)

	if len(anime.Data) > 0 {
		if len(anime.Data[0].Attributes.Titles.En) <= 0 {
			anime.Data[0].Attributes.Titles.En = anime.Data[0].Attributes.Titles.EnJp
		}

		anime.Data[0].Attributes.Synopsis = strings.Join(strings.Fields(anime.Data[0].Attributes.Synopsis), " ")
		anime.Data[0].Attributes.Synopsis = strings.Replace(anime.Data[0].Attributes.Synopsis, "\"", "\\\"", -1)
	}
	fmt.Println(*anime)
}

/**
 Create an embed view and fill it with the anime infos. Then send it to the channel
 */
func createAndSendEmbedMessage(session *discordgo.Session, channelId string, anime AnimeInfos) {
	var embed discordgo.MessageEmbed

	jsonBody := fmt.Sprintf(`{
    "title": "**%s** *(%s)*" ,
    "description": "%s (%d episode(s)) - %s ` + "```%s```" +  `",` +
    //"url": "https://myanimelist.net/", //TODO Find a way to get the MAL page of the anime
    `"color": 8311585,

    "thumbnail": {
      "url": "%s"
    },

    "author": {
      "name": "LuluBot",
      "icon_url": "https://i.imgur.com/gHmi1FU.jpg"
    }

  }`, anime.Attributes.Titles.En, anime.Attributes.Titles.JaJp,
		anime.Attributes.Subtype, anime.Attributes.EpisodeCount,
		anime.Attributes.Status, anime.Attributes.Synopsis, anime.Attributes.PosterImage.Original)

	json.Unmarshal([]byte(jsonBody), &embed)

	_, err := session.ChannelMessageSendEmbed(channelId, &embed)
	if err != nil {
		fmt.Println("Error while sending message,", err)
	}
}