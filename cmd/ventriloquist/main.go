package main

import (
	"context"
	"os"

	"github.com/Xe/ln"
	"github.com/asdine/storm"
	"github.com/bwmarrin/discordgo"
	"github.com/joeshaw/envdecode"
	_ "github.com/joho/godotenv/autoload"
	bbot "github.com/withinsoft/ventriloquist/internal/bot"
)

type config struct {
	DiscordToken string `env:"DISCORD_TOKEN,required"`
	DBPath       string `env:"DB_PATH,default=var/vent.db"`
	AdminRole    string `env:"ADMIN_ROLE,required"`
}

func main() {
	ctx := context.Background()
	ctx = ln.WithF(ctx, ln.F{
		"in": "main",
	})

	_ = os.MkdirAll("var", 0700)

	var cfg config
	err := envdecode.StrictDecode(&cfg)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	ln.Log(ctx, ln.Action("discordgo session created"))

	db, err := storm.Open(cfg.DBPath)
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	ln.Log(ctx, ln.Action("database opened"))

	b := bot{
		cfg: cfg,
		db:  DB{s: db},
		dg:  dg,
	}
	must := func(err error) {
		if err != nil {
			ln.FatalErr(ctx, err)
		}
	}
	cs := bbot.NewCommandSet()
	cs.Prefix = ";"

	must(cs.AddCmd("add", "adds a systemmate to the list of proxy tags", bbot.NoPermissions, b.addSystemmate))
	must(cs.AddCmd("list", "lists systemmates", bbot.NoPermissions, b.listSystemmates))
	must(cs.AddCmd("update", "updates systemmates avatars and optionally name", bbot.NoPermissions, b.updateAvatar))
	must(cs.AddCmd("del", "removes a systemmate", bbot.NoPermissions, b.delSystemmate))
	must(cs.AddCmd("nuke", "removes all system data", bbot.NoPermissions, b.nukeSystem))
	must(cs.AddCmd("chproxy", "changes proxy method for a systemmate", bbot.NoPermissions, b.changeProxy))
	must(cs.AddCmd("mod_list", "mod: lists systemmates for a user", b.modOnly, b.modForce(
		"list",
		"usage: ;mod_list <mention the user>\n\n(don't include the angle brackets)",
		2,
		b.listSystemmates,
	)))
	must(cs.AddCmd("mod_del", "mod: removes a systemmate for a user", b.modOnly, b.modForce(
		"del",
		"usage: ;mod_del <mention the user> <name>\n\n(don't include the angle brackets)",
		3,
		b.delSystemmate,
	)))
	must(cs.AddCmd("mod_update", "mod: removes a systemmate for a user", b.modOnly, b.modForce(
		"update",
		"usage: ;mod_update <mention the user> <name> <new avatar url> <new name>\n\n(don't include the angle brackets)",
		5,
		b.updateAvatar,
	)))
	must(cs.AddCmd("mod_chproxy", "mod: changes proxy method of a systemmate for a user", b.modOnly, b.modForce(
		"update",
		"usage: ;mod_chproxy <mention the user>\n\n(don't include the angle brackets)",
		999,
		b.changeProxy,
	)))
	ln.Log(ctx, ln.Action("added commands to mux"))

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		err := cs.Run(s, m.Message)
		if err != nil {
			ln.Error(context.Background(), err)
		}
	})
	dg.AddHandler(b.proxyScrape)
	ln.Log(ctx, ln.Action("added discordgo handlers"))

	err = dg.Open()
	if err != nil {
		ln.FatalErr(ctx, err)
	}
	ln.Log(ctx, ln.Action("opened discordgo websocket"))

	ln.Log(ctx, ln.Action("waiting forever (and a day)"))
	for {
		select {}
	}
}
