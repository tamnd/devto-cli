package devto

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

type Domain struct{}

func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "devto",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "devto",
			Short:  "A command line for DEV Community.",
			Long: `A command line for DEV Community (dev.to).

Browse top articles, filter by tag, or list articles by user.
No API key required.`,
			Site: "https://" + Host,
			Repo: "https://github.com/tamnd/devto-cli",
		},
	}
}

func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "top", Group: "articles", List: true,
		URIType: "article", Summary: "Top DEV.to articles"}, topArticles)

	kit.Handle(app, kit.OpMeta{Name: "tag", Group: "articles", List: true,
		URIType: "article", Summary: "DEV.to articles by tag",
		Args: []kit.Arg{{Name: "tag", Help: "tag name"}}}, tagArticles)

	kit.Handle(app, kit.OpMeta{Name: "user", Group: "articles", List: true,
		URIType: "article", Summary: "DEV.to articles by user",
		Args: []kit.Arg{{Name: "username", Help: "DEV.to username"}}}, userArticles)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClientWithConfig(DefaultConfig())
	if cfg.UserAgent != "" {
		c.cfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.cfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.cfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.cfg.Timeout = cfg.Timeout
		c.http.Timeout = cfg.Timeout
	}
	return c, nil
}

type listIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type tagIn struct {
	Tag    string  `kit:"arg" help:"tag name"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type userIn struct {
	Username string  `kit:"arg" help:"DEV.to username"`
	Limit    int     `kit:"flag,inherit" help:"max results"`
	Client   *Client `kit:"inject"`
}

func topArticles(ctx context.Context, in listIn, emit func(*Article) error) error {
	articles, err := in.Client.Top(ctx, in.Limit)
	if err != nil {
		return err
	}
	for _, a := range articles {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func tagArticles(ctx context.Context, in tagIn, emit func(*Article) error) error {
	if in.Tag == "" {
		return errs.Usage("tag name required")
	}
	articles, err := in.Client.ByTag(ctx, in.Tag, in.Limit)
	if err != nil {
		return err
	}
	for _, a := range articles {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func userArticles(ctx context.Context, in userIn, emit func(*Article) error) error {
	if in.Username == "" {
		return errs.Usage("username required")
	}
	articles, err := in.Client.ByUser(ctx, in.Username, in.Limit)
	if err != nil {
		return err
	}
	for _, a := range articles {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}
