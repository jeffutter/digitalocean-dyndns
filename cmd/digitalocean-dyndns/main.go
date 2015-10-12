package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"os"
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

type context struct {
	DoClient *godo.Client
}

type ipResponse struct {
	IP string `json:"ip"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err.Error())
	}

	pat := os.Getenv("access_token")
	tokenSource := &TokenSource{
		AccessToken: pat,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	c := context{
		DoClient: client,
	}

	app := cli.NewApp()
	app.Name = "digitalocean-dyndns"
	app.Author = "Jeffery Utter"
	app.Email = "jeff@jeffutter.com"
	app.Commands = []cli.Command{
		{
			Name:      "getip",
			ShortName: "g",
			Usage:     "getip",
			Action:    getIp(c),
		},
		{
			Name:      "update",
			ShortName: "u",
			Usage:     "update",
			Action:    updateIp(c),
		},
	}
	app.Run(os.Args)
}

func perr(err error) {
	if err != nil {
		panic(err)
	}
}

func GetIP() string {
	url := "https://api.ipify.org/?format=json"

	res, err := http.Get(url)
	perr(err)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	perr(err)
	var data ipResponse
	if err = json.Unmarshal(body, &data); err != nil {
		panic(err)
	}
	return data.IP
}

func getIp(context context) func(ctx *cli.Context) {
	return func(ctx *cli.Context) {
		ip := GetIP()
		fmt.Println(ip)
	}
}

func getRecordId(context context) (int, error) {

	opt := &godo.ListOptions{}

	records, _, err := context.DoClient.Domains.Records(os.Getenv("domain"), opt)
	if err != nil {
		return 0, err
	}
	for _, record := range records {
		if record.Name == os.Getenv("host") {
			return record.ID, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("Host %s, not found on domain %s.", os.Getenv("host"), os.Getenv("domain")))
}

func updateIp(context context) func(ctx *cli.Context) {
	return func(ctx *cli.Context) {
		ip := GetIP()

		id, err := getRecordId(context)
		perr(err)
		fmt.Printf("Domain ID: %d\n", id)

		editRequest := &godo.DomainRecordEditRequest{
			Data: ip,
		}

		domainRecord, response, err := context.DoClient.Domains.EditRecord(os.Getenv("domain"), id, editRequest)
		perr(err)
		fmt.Printf("%+v\n", response)
		fmt.Printf("%+v\n", domainRecord)
	}
}
