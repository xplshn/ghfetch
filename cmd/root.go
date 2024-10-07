package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TheZoraiz/ascii-image-converter/aic_package"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
)

type gitHubUser struct {
	Login                string `json:"login"`
	Name                 string `json:"name"`
	Repos                int    `json:"public_repos"`
	Followers            int    `json:"followers"`
	Following            int    `json:"following"`
	TotalStarsEarned     int    `json:"total_star_earned"`
	TotalCommitsThisYear int    `json:"total_commit_this_year"`
	TotalPRs             int    `json:"total_pr"`
	TotalIssues          int    `json:"total_issues"`
}

type RepoInfo struct {
	Author         string   `json:"owner.login"`
	Description    string   `json:"description"`
	Language       string   `json:"language"`
	License        string   `json:"license.name"`
	LastUpdated    string   `json:"updated_at"`
	Version        string   `json:"latest_release.tag_name"`
	Released       string   `json:"latest_release.published_at"`
	OwnerAvatar    string   `json:"owner.avatar_url"`
	Stars          int      `json:"stargazers_count"`
	Topics         []string `json:"topics"`
}

var (
	username       string
	highlightColor string
	accessToken    string
	repoURL        string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ghfetch",
	Short: "Fetch GitHub user's profile, just like neofetch",
	RunE: func(cmd *cobra.Command, args []string) error {
		if accessToken == "" {
			accessToken = os.Getenv("GH_TOKEN")
			if accessToken == "" {
				accessToken = os.Getenv("GITHUB_TOKEN")
			}
		}
		if accessToken == "" {
			return errors.New("access token environment variable is not set")
		}
		s := spinner.New(spinner.CharSets[2], 50*time.Millisecond)
		_ = s.Color("blue")
		if highlightColor != "" {
			_ = s.Color(highlightColor)
		}
		s.Start()
		paneWidth := getTermWidth() / 2
		defaultWidth := 50
		defaultHeight := 25
		scaleFactor := float64(paneWidth) / float64(defaultWidth)
		if scaleFactor > 1 {
			scaleFactor = 1
		}
		newWidth := int(float64(defaultWidth) * scaleFactor)

		if username == "" && repoURL == "" {
			return errors.New("please provide a github username using the --user flag or a repository URL using the --repo flag")
		}
		flags := aic_package.DefaultFlags()
		flags.Dimensions = []int{newWidth, int(float64(defaultHeight) * scaleFactor)}
		flags.Colored = true
		flags.CustomMap = " .-=+#@"

		var asciiArt string
		var err error
		if username != "" {
			asciiArt, err = aic_package.Convert(fmt.Sprintf("https://github.com/%s.png", username), flags)
		} else if repoURL != "" {
			repoInfo, err := fetchRepoInfo(repoURL, accessToken)
			if err != nil {
				return errors.New("error fetching repository information")
			}
			asciiArt, err = aic_package.Convert(repoInfo.OwnerAvatar, flags) // Fetch author's image
		}
		if err != nil {
			return err
		}
		leftPane := lipgloss.NewStyle().Width(newWidth).Render(asciiArt)

		var user *gitHubUser
		if username != "" {
			user, err = fetchUserWithGraphQL(username, accessToken)
			if err != nil {
				return errors.New("error fetching user information")
			}
		}
		s.Stop()

		titleColor := colorMap[highlightColor].SprintFunc()
		User := titleColor("User")
		Name := titleColor("Name")
		Repos := titleColor("Repos")
		Followers := titleColor("Followers")
		Following := titleColor("Following")
		TotalStarsEarned := titleColor("Total Stars Earned")
		TotalCommitsThisYear := titleColor("Total Commits This Year")
		TotalPRs := titleColor("Total PRs")
		TotalIssues := titleColor("Total Issues")

		userInfoPane := []string{}
		if username != "" {
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %s", User, username))
			userInfoPane = append(userInfoPane, separator(username))
			if user.Name != "" {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %s", Name, user.Name))
			}
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", Repos, user.Repos))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", Followers, user.Followers))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", Following, user.Following))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", TotalStarsEarned, user.TotalStarsEarned))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", TotalCommitsThisYear, user.TotalCommitsThisYear))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", TotalPRs, user.TotalPRs))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  %s: %d", TotalIssues, user.TotalIssues))
		}

		if repoURL != "" {
			repoInfo, err := fetchRepoInfo(repoURL, accessToken)
			if err != nil {
				return errors.New("error fetching repository information")
			}
			if username != "" {
				userInfoPane = append(userInfoPane, separator(username))
			}
			userInfoPane = append(userInfoPane, fmt.Sprintf("  Repo URL: %s", repoURL))
			userInfoPane = append(userInfoPane, fmt.Sprintf("  Author: %s", repoInfo.Author))
			if repoInfo.Description != "" {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  Description: %s", repoInfo.Description))
			}
			if repoInfo.Language != "" {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  Language: %s", repoInfo.Language))
			}
			if repoInfo.License != "" {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  License: %s", repoInfo.License))
			}
			userInfoPane = append(userInfoPane, fmt.Sprintf("  Last Updated: %s", repoInfo.LastUpdated))
			if repoInfo.Version != "" {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  Version: %s", repoInfo.Version))
			}
			if repoInfo.Released != "" {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  Released: %s", repoInfo.Released))
			}
			userInfoPane = append(userInfoPane, fmt.Sprintf("  Stars: %d", repoInfo.Stars))
			if len(repoInfo.Topics) > 0 {
				userInfoPane = append(userInfoPane, fmt.Sprintf("  Topics: %v", repoInfo.Topics))
			}
		}

		userInfoPane = append(userInfoPane, separator(username))
		userInfoPane = append(userInfoPane, getPalette())
		rightPaneContent := strings.Join(userInfoPane, "\n")
		rightPane := lipgloss.NewStyle().Width(paneWidth).Render(rightPaneContent)

		fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, leftPane, rightPane))
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&username, "user", "u", "", "GitHub username")
	rootCmd.Flags().StringVarP(&highlightColor, "color", "c", "blue", "Highlight color red, green, yellow, blue, magenta, cyan")
	rootCmd.Flags().StringVar(&accessToken, "access-token", "", "Your GitHub access token")
	rootCmd.Flags().StringVar(&repoURL, "repo", "", "Repository URL")
}

var colorMap = map[string]*color.Color{
	"red":     color.New(color.FgRed),
	"green":   color.New(color.FgGreen),
	"yellow":  color.New(color.FgYellow),
	"blue":    color.New(color.FgBlue),
	"magenta": color.New(color.FgMagenta),
	"cyan":    color.New(color.FgCyan),
}

func getTermWidth() int {
	err := termbox.Init()
	if err != nil {
		log.Fatalf("error initializing termbox: %v", err)
	}
	defer termbox.Close()

	width, _ := termbox.Size()
	return width
}

func separator(value string) string {
	titleColor := colorMap[highlightColor].SprintFunc()
	lineLength := 6 + len(value)
	return titleColor("  " + strings.Repeat("-", lineLength))
}

func fetchUserWithGraphQL(username, accessToken string) (*gitHubUser, error) {
	url := "https://api.github.com/graphql"
	token := accessToken
	var endCursor string
	hasNextPage := true
	var totalStars int
	var totalRepos int
	var query string

	for hasNextPage {
		if endCursor == "" {
			query = fmt.Sprintf(`{
		user(login: "%s") {
			name
			repositories(first: 100, ownerAffiliations: OWNER, isFork: false, orderBy: {direction: DESC, field: STARGAZERS}) {
				nodes {
					stargazers {
						totalCount
					}
				}
				pageInfo {
					endCursor
					hasNextPage
				}
			}
			followers {
				totalCount
			}
			following {
				totalCount
			}
			starredRepositories {
				totalCount
			}
			contributionsCollection {
				totalCommitContributions
			}
			pullRequests {
				totalCount
			}
			issues {
				totalCount
			}
		}
	}`, username)
		} else {
			query = fmt.Sprintf(`{
		user(login: "%s") {
			name
			repositories(first: 100, ownerAffiliations: OWNER, isFork: false, orderBy: {direction: DESC, field: STARGAZERS}, after: "%s") {
				nodes {
					stargazers {
						totalCount
					}
				}
				pageInfo {
					endCursor
					hasNextPage
				}
			}
			followers {
				totalCount
			}
			following {
				totalCount
			}
			starredRepositories {
				totalCount
			}
			contributionsCollection {
				totalCommitContributions
			}
			pullRequests {
				totalCount
			}
			issues {
				totalCount
			}
		}
	}`, username, endCursor)
		}

		body, err := json.Marshal(map[string]string{
			"query": query,
		})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		for _, repo := range result.Data.User.Repositories.Nodes {
			totalStars += repo.Stargazers.TotalCount
		}
		totalRepos += len(result.Data.User.Repositories.Nodes)
		endCursor = result.Data.User.Repositories.PageInfo.EndCursor
		hasNextPage = result.Data.User.Repositories.PageInfo.HasNextPage
	}
	user := &gitHubUser{
		Name:                 result.Data.User.Name,
		Repos:                totalRepos,
		Followers:            result.Data.User.Followers.TotalCount,
		Following:            result.Data.User.Following.TotalCount,
		TotalStarsEarned:     totalStars,
		TotalCommitsThisYear: result.Data.User.ContributionsCollection.TotalCommitContributions,
		TotalPRs:             result.Data.User.PullRequests.TotalCount,
		TotalIssues:          result.Data.User.Issues.TotalCount,
	}

	return user, nil
}

func fetchRepoInfo(repoURL, accessToken string) (*RepoInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", strings.TrimPrefix(repoURL, "https://github.com/"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var repoInfo RepoInfo
	err = json.NewDecoder(resp.Body).Decode(&repoInfo)
	if err != nil {
		return nil, err
	}

	return &repoInfo, nil
}

var result struct {
	Data struct {
		User struct {
			Name      string `json:"name"`
			Followers struct {
				TotalCount int `json:"totalCount"`
			}
			Following struct {
				TotalCount int `json:"totalCount"`
			}
			Repositories struct {
				Nodes []struct {
					Stargazers struct {
						TotalCount int `json:"totalCount"`
					} `json:"stargazers"`
				} `json:"nodes"`
				PageInfo struct {
					EndCursor   string `json:"endCursor"`
					HasNextPage bool   `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"repositories"`
			PullRequests struct {
				TotalCount int `json:"totalCount"`
			}
			Issues struct {
				TotalCount int `json:"totalCount"`
			}
			ContributionsCollection struct {
				TotalCommitContributions int `json:"totalCommitContributions"`
			}
		} `json:"user"`
	} `json:"data"`
}

func getPalette() string {
	color1 := "\x1b[0;29m███\x1b[0m"
	color2 := "\x1b[0;31m███\x1b[0m"
	color3 := "\x1b[0;32m███\x1b[0m"
	color4 := "\x1b[0;33m███\x1b[0m"
	color5 := "\x1b[0;34m███\x1b[0m"
	color6 := "\x1b[0;35m███\x1b[0m"
	color7 := "\x1b[0;36m███\x1b[0m"
	color8 := "\x1b[0;37m███\x1b[0m"

	return color1 + color2 + color3 + color4 + color5 + color6 + color7 + color8
}
