package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection

func init() {

}

func main() {
	//goals:
	//1. subcribe 1 tai khoan twitter
	//2. bot gui tin nhan khi co tweet moi
	//3. cac tin nhan cu se bi luu vao db
	//4. dung signals de stop bot
	//5. dung signals de restart bot
	//6. đánh dấu tweet đã được gửi bằng cách thêm 1 field bool vào db, khởi tạo mặc định -> false, khi gửi xong thì update -> true
	// go scraper("Icetea_Labs")
	// time.Sleep(time.Second * 5)

	go sendMessage()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}
func sendMessage() {
	mongoclient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		fmt.Println(err)
	}
	collection = mongoclient.Database("tracking").Collection("tweets")
	collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "username", Value: 1},
			{Key: "URLs", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	bot, err := tgbotapi.NewBotAPI("5516388529:AAHpOxW2utxG9A-AmmuEvFE24m-VBetXb3Q")
	if err != nil {
		fmt.Println(err)
		return
	}
	bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	// u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	if err != nil {
		fmt.Println(err)
		return
	}
	for update := range updates {
		if update.Message != nil { // If we got a message
			// log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			// go scraper(update.Message.Text)
			for {
				tweetURLs := scraper(update.Message.Text)
				for _, tweetURL := range tweetURLs {
					collection.InsertOne(context.Background(), bson.D{
						{Key: "username", Value: update.Message.Text},
						{Key: "URLs", Value: tweetURL},
						{Key: "isSent", Value: false},
					})
					//check if tweet is sent
					var result bson.M
					err := collection.FindOne(context.Background(), bson.D{
						{Key: "username", Value: update.Message.Text},
						{Key: "URLs", Value: tweetURL},
					}).Decode(&result)
					if err != nil {
						fmt.Println(err)
					}
					if result["isSent"] == false {
						bot.Send(tgbotapi.NewMessage(1262995839, tweetURL))
						collection.UpdateOne(context.Background(), bson.D{
							{Key: "username", Value: update.Message.Text},
							{Key: "URLs", Value: tweetURL},
						}, bson.D{
							{Key: "$set", Value: bson.D{
								{Key: "isSent", Value: true},
							}},
						})
					}
					//if there is no new tweet, sleep 5 seconds then continue loop
				}

			}
		}
	}
}
func scraper(username string) []string {
	scraper := twitterscraper.New()
	//truyen param == username vao bot telegram -> truyen vao trong gettweets de lay ra URLs, luu vao db ? de check xem no co ton tai hay chua.
	//neu url khong ton` tai, bot -> message. Sleep 30phut -> 50 newest tweets.
	//lam sao de subcribe cung 1 luc nhieu tai khoan twitter?
	//tweet duoc lay thu tu: newest -> lastest
	var tweetURLs []string
	for tweet := range scraper.GetTweets(context.Background(), username, 20) {
		if tweet.Error != nil {
			fmt.Println(tweet.Error)
			continue
		}
		// bot.Send(tgbotapi.NewMessage(1262995839, tweet.PermanentURL))
		tweetURLs = append(tweetURLs, tweet.PermanentURL)
	}
	return tweetURLs
}

//13:51 20/10/2022: đã filter ra được msg gửi và chưa gửi. vấn đề còn tồn đọng: lấy tweet mới nhất và gửi (real time), chưa implement go routines -> chưa subscribe nhiều tài khoản twitter cùng lúc
