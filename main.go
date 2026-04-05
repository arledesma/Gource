package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/arledesma/gource-tui/config"
	"github.com/arledesma/gource-tui/model"
	"github.com/arledesma/gource-tui/parser"
)

var version = "dev"

func main() {
	// Subprocess mode for terminal detection — must run before cobra
	if len(os.Args) == 2 && os.Args[1] == "--detect-term" {
		model.RunDetectSubprocess()
		return
	}

	cfg := config.DefaultSettings()

	rootCmd := &cobra.Command{
		Use:   "gource-tui [path]",
		Short: "Visualize source control history as an animated graph in the terminal",
		Long: `gource-tui renders git repository history as a force-directed graph
with bloom effects, using sixel graphics in your terminal.

Requires a sixel-capable terminal (WezTerm, Windows Terminal 1.22+, foot, etc.)`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cfg.Path = args[0]
			}

			var parserOpts parser.Options
			if !cfg.StartDate.IsZero() {
				parserOpts.StartDate = cfg.StartDate.Format("2006-01-02")
			}
			if !cfg.StopDate.IsZero() {
				parserOpts.StopDate = cfg.StopDate.Format("2006-01-02")
			}

			p, err := parser.New(cfg.Path, parserOpts)
			if err != nil {
				return fmt.Errorf("cannot open %s: %w", cfg.Path, err)
			}

			// Detect terminal pixel size before Bubble Tea takes stdin
			termSize := model.DetectTermPixelSize()
			cfg.DetectedCellW = termSize.CellW
			cfg.DetectedCellH = termSize.CellH
			cfg.DetectedPixW = termSize.PixW
			cfg.DetectedPixH = termSize.PixH

			m := model.New(cfg, p)
			prog := tea.NewProgram(m)

			if _, err := prog.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	flags := rootCmd.Flags()
	flags.Float64VarP(&cfg.DaysPerSecond, "speed", "s", cfg.DaysPerSecond, "Days of history per second of playback")
	flags.Float64Var(&cfg.AutoSkip, "auto-skip", cfg.AutoSkip, "Auto-skip idle periods longer than N days")
	flags.Float64Var(&cfg.FileIdleTime, "file-idle-time", cfg.FileIdleTime, "Seconds before idle files fade")
	flags.Float64Var(&cfg.UserIdleTime, "user-idle-time", cfg.UserIdleTime, "Seconds before idle users disappear")
	flags.BoolVar(&cfg.Loop, "loop", false, "Loop playback when finished")
	flags.BoolVar(&cfg.NoBloom, "no-bloom", false, "Disable bloom post-processing (faster)")
	flags.StringVar(&cfg.UserFilter, "user-filter", "", "Regex to filter users (only show matching)")
	flags.StringVar(&cfg.FileFilter, "file-filter", "", "Regex to filter file paths")
	flags.StringVar(&cfg.Background, "background", "", "Background color as hex (e.g. #1a1a2e)")
	flags.BoolVar(&cfg.HideFilenames, "hide-filenames", false, "Hide file name labels")
	flags.BoolVar(&cfg.HideDirnames, "hide-dirnames", false, "Hide directory name labels")
	flags.BoolVar(&cfg.HideUsernames, "hide-usernames", false, "Hide user name labels")
	flags.BoolVar(&cfg.HideDate, "hide-date", false, "Hide the date overlay")
	flags.BoolVar(&cfg.HideProgress, "hide-progress", false, "Hide the progress bar")
	flags.BoolVar(&cfg.Debug, "debug", false, "Show FPS counter and debug info")
	flags.StringVar(&cfg.CellSize, "cell-size", "", "Override cell pixel size as WxH (e.g. 8x18)")
	flags.StringVar(&cfg.Theme, "theme", "dark", "Color theme (dark, light, solarized, monokai)")
	flags.Float64Var(&cfg.RenderScale, "scale", 1.0, "Render resolution scale (0.5 = half res, faster)")
	flags.IntVar(&cfg.FPS, "fps", 30, "Target frames per second (lower = less CPU)")

	var startDate, stopDate string
	flags.StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD)")
	flags.StringVar(&stopDate, "stop-date", "", "Stop date (YYYY-MM-DD)")

	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if startDate != "" {
			t, err := time.Parse("2006-01-02", startDate)
			if err != nil {
				return fmt.Errorf("invalid --start-date: %w", err)
			}
			cfg.StartDate = t
		}
		if stopDate != "" {
			t, err := time.Parse("2006-01-02", stopDate)
			if err != nil {
				return fmt.Errorf("invalid --stop-date: %w", err)
			}
			cfg.StopDate = t
		}
		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
