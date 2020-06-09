package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fatih/color"
	"github.com/go-yaml/yaml"
)

func logError(err string) {
	color.Red("[Errors] %s : %s\n", time.Now().String, err)
}

func logOK(str string) {
	color.Green("[Success] %s : %s\n", time.Now(), str)
}

func logPrint(str string) {
	color.Blue("%s\n", str)
}

type Config struct {
	Token string
	Maps  []Map
}

type Map struct {
	Activator string
	Reply     string
	Ignored   []string `yaml:",flow"`
}

var universalConfig *Config

func input(reader *bufio.Reader) string {
	raw, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Something went wrong when reading stdin: " + err.Error())
		return ""
	}
	switch runtime.GOOS {
	case "windows":
		return strings.Replace(raw, "\r\n", "", -1)
	default:
		return strings.Replace(raw, "\n", "", 1)
	}
}

func readFile(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		logError("File not found! Creating one in chosen directory...")
		file, err = createFile(path)
		if err != nil {
			logError("Could not create file." + err.Error())
			return nil
		}
		defaultYaml(path)
		logOK("File created successfully.")
	}
	defer file.Close()
	reader, err := ioutil.ReadAll(file)
	if err != nil {
		logError("Could not read file! :" + err.Error())
		return nil
	}
	return reader
}

func writeFile(path string, data []byte) {
	ioutil.WriteFile(path, data, 0644)
}

func defaultYaml(path string) {
	newYaml, err := yaml.Marshal(universalConfig)
	if err != nil {
		logError("Could not create default YAML.")
		return
	}

	writeFile(path, newYaml)
}

func createFile(path string) (*os.File, error) {
	file, err := os.Create(fmt.Sprint(path))
	return file, err
}

func readConfig(path string) {
	universalConfig = new(Config)
	err := yaml.Unmarshal(readFile(path), &universalConfig)
	if err != nil {
		logError("Could not read config file.")
		return
	}
}

func writeConfig(path string, uc *Config) {
	data, err := yaml.Marshal(&uc)
	if err != nil {
		logError("Could not save to config.")
		return
	}
	writeFile(path, data)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	logPrint("Welcome to General Purpose Discord Sniper")
	logPrint("Please input config file path. If none is specified, the default path will be ./config.yaml . In case it doesn't exist, it will be created.")
	path := input(reader)
	if path == "" {
		path = "config.yaml"
	}
	readConfig(path)
	logOK("Configuration loaded successfully.")
	if universalConfig.Maps == nil {
		logError("No events found.")
	} else {
		client.AddHandler(sniper)
	}
	token := universalConfig.Token
	if token == "" {
		logPrint("Token is empty. Please input your discord token to proceed.")
		token = input(reader)
		universalConfig.Token = token
		writeConfig(path, universalConfig)
	}
	client, err := discordgo.New(token)
	if err != nil {
		logError("Could not create session with token.")
		return
	}
	err = client.Open()
	if err != nil {
		logError("Could not open session.")
		return
	}
	logOK(fmt.Sprint("Welcome to GPDS, ", client.State.User.String(), "!"))
	if universalConfig.Maps == nil {
		logPrint("Please insert some events:")
		addMultiple(reader, client)
		writeConfig(path, universalConfig)
		logOK("Success adding events. Please restart the sniper to save changes.")
		return
	}
	logOK("Listening to events...")
	logPrint("Press Ctrl+C to quit.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-quit
	logOK("Quitting GPDS...")
	client.Close()
}

func eventSuccessString(client *discordgo.Session, message *discordgo.MessageCreate) string {
	guild, _ := client.Guild(message.GuildID)
	return guild.Name
}

func sniper(client *discordgo.Session, message *discordgo.MessageCreate) {
	for _, event := range universalConfig.Maps {
		for _, ID := range event.Ignored {
			if ID == message.GuildID {
				logPrint(fmt.Sprint("Ignoring event ", event.Activator, " in ", eventSuccessString(client, message)))
				return
			}
		}
		if strings.Contains(message.Content, event.Activator) {
			client.ChannelMessageSend(message.ChannelID, event.Reply)
			logOK(fmt.Sprint("Sniped event: ", event.Activator, " in ", eventSuccessString(client, message), " with reply: ", event.Reply))
		}
	}
}

func addMultiple(reader *bufio.Reader, client *discordgo.Session) {
	logPrint("Do you want to add another event? y/n")
	option := input(reader)
	switch strings.ToLower(option) {
	case "y":
		snipeAdd(reader, client)
		addMultiple(reader, client)
	case "n":
		return
	}
}

func snipeAdd(reader *bufio.Reader, client *discordgo.Session) {
	logPrint("Please input the message the bot will reply to:")
	message := input(reader)
	logPrint("Please input the reply to: " + message)
	reply := input(reader)
	logPrint("Please input guilds that this snipe will be ignored in, separated by a comma , ")
	joinGuilds := input(reader)
	guilds := strings.Split(joinGuilds, ",")
	checkedGuilds := checkGuilds(client, guilds)
	addEntry(message, reply, checkedGuilds)
}

func guildsToString(guilds []*discordgo.Guild) []string {
	var ids []string
	for _, guild := range guilds {
		ids = append(ids, guild.ID)
	}
	return ids
}

func addEntry(activator string, reply string, guilds []*discordgo.Guild) {
	newMap := Map{Activator: activator, Reply: reply, Ignored: guildsToString(guilds)}
	universalConfig.Maps = append(universalConfig.Maps, newMap)
}

func checkGuilds(client *discordgo.Session, guildIDS []string) []*discordgo.Guild {
	var guilds []*discordgo.Guild
	for _, guildID := range guildIDS {
		guild, err := client.Guild(guildID)
		if err != nil {
			continue
		}
		guilds = append(guilds, guild)
	}
	return guilds
}
