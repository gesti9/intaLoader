package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ExtractAndRun извлекает shortcode из ссылки Instagram и выполняет команду instaloader с этим shortcode.
func ExtractAndRun(instagramURL string) (string, error) {
	// Извлечение shortcode
	u, err := url.Parse(instagramURL)
	if err != nil {
		return "", fmt.Errorf("ошибка разбора URL: %v", err)
	}

	pathSegments := strings.Split(u.Path, "/")
	if len(pathSegments) < 3 {
		return "", fmt.Errorf("не удалось найти shortcode в ссылке")
	}

	shortcode := pathSegments[2]

	// Выполнение команды instaloader
	cmd := exec.Command("instaloader", "--", "-"+shortcode)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Путь к папке с загруженными файлами
	folderPath := filepath.Join(".", shortcode)
	return folderPath, nil
}

func findVideoFile(folderPath string) (string, error) {
	var videoFile string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mp4") {
			videoFile = path
			return filepath.SkipDir // Останавливаем обход после нахождения первого видеофайла
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if videoFile == "" {
		return "", fmt.Errorf("видео не найдено в папке: %s", folderPath)
	}

	return videoFile, nil
}

func isValidURL(s string) bool {
	parsedURL, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	// Простейшая проверка на то, что URL имеет схему (например, http или https).
	return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
}

func handleRequest(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if isValidURL(update.Message.Text) {
		folderPath, err := ExtractAndRun(update.Message.Text)
		folderPath = "-" + folderPath
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка загрузки: "+err.Error())
			bot.Send(msg)

		}
		fmt.Println("файл находится " + folderPath)
		videoFile, err := findVideoFile(folderPath)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска видео: "+err.Error())
			bot.Send(msg)

		}

		// Используем путь к видео для создания конфигурации видео-сообщения
		videoMsg := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FilePath(videoFile))
		bot.Send(videoMsg)

		// Удаляем папку после успешной отправки видео
		if err := os.RemoveAll(folderPath); err != nil {
			log.Printf("Ошибка удаления папки: %s, %v", folderPath, err)
		} else {
			log.Printf("Папка успешно удалена: %s", folderPath)
		}

	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте ссылку!")
		msg.ReplyToMessageID = update.Message.MessageID
		bot.Send(msg)
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI("7318293403:AAEkBKhZWIYvINRFXEjq-1ZrMJBsOXEcuDY")
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			go handleRequest(bot, update)
		}
	}
}
