// Package apps defines curated "app bundles" — a named service mapped to the
// set of domains it uses — so the dashboard can block a whole app with one
// click. Focus Mode, Doom-scroll Guard, and the Screen-Time Bank all reuse
// these bundles.
package apps

// Bundle is one blockable app/service.
type Bundle struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Emoji    string   `json:"emoji"`
	Category string   `json:"category"` // social | video | gaming | gambling
	Domains  []string `json:"domains"`
}

// All bundles, in display order.
var All = []Bundle{
	{ID: "tiktok", Name: "TikTok", Emoji: "🎵", Category: "social",
		Domains: []string{"tiktok.com", "tiktokcdn.com", "tiktokv.com", "byteoversea.com", "musical.ly", "ibytedtos.com", "tiktokcdn-us.com"}},
	{ID: "instagram", Name: "Instagram", Emoji: "📸", Category: "social",
		Domains: []string{"instagram.com", "cdninstagram.com", "instagramstatic-a.akamaihd.net"}},
	{ID: "youtube", Name: "YouTube", Emoji: "▶️", Category: "video",
		Domains: []string{"youtube.com", "youtu.be", "ytimg.com", "googlevideo.com", "youtubei.googleapis.com", "yt3.ggpht.com"}},
	{ID: "facebook", Name: "Facebook", Emoji: "👤", Category: "social",
		Domains: []string{"facebook.com", "fbcdn.net", "fb.com", "facebook.net", "fbsbx.com"}},
	{ID: "twitter", Name: "X / Twitter", Emoji: "🐦", Category: "social",
		Domains: []string{"twitter.com", "x.com", "twimg.com", "t.co"}},
	{ID: "reddit", Name: "Reddit", Emoji: "👽", Category: "social",
		Domains: []string{"reddit.com", "redd.it", "redditstatic.com", "redditmedia.com"}},
	{ID: "snapchat", Name: "Snapchat", Emoji: "👻", Category: "social",
		Domains: []string{"snapchat.com", "sc-cdn.net", "snap.com"}},
	{ID: "netflix", Name: "Netflix", Emoji: "🎬", Category: "video",
		Domains: []string{"netflix.com", "nflxvideo.net", "nflximg.net", "nflxext.com", "nflxso.net"}},
	{ID: "twitch", Name: "Twitch", Emoji: "🎮", Category: "video",
		Domains: []string{"twitch.tv", "ttvnw.net", "jtvnw.net"}},
	{ID: "gaming", Name: "Gaming", Emoji: "🕹️", Category: "gaming",
		Domains: []string{"steampowered.com", "steamcommunity.com", "epicgames.com", "roblox.com", "rbxcdn.com", "minecraft.net", "ea.com", "riotgames.com"}},
	{ID: "gambling", Name: "Gambling", Emoji: "🎰", Category: "gambling",
		Domains: []string{"bet365.com", "pokerstars.com", "draftkings.com", "fanduel.com", "williamhill.com", "888casino.com"}},
}

// ByID returns the bundle with the given id, or nil.
func ByID(id string) *Bundle {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}

// FocusDefault is the set of app IDs blocked by Focus Mode (distractions).
var FocusDefault = []string{"tiktok", "instagram", "youtube", "facebook", "twitter", "reddit", "snapchat", "netflix", "twitch", "gaming"}
