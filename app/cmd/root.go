package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Ingresso CLI",
	Long:  `Ingresso CLI Version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ingresso CLI v0.2 -- HEAD")
	},
}

var rootCmd = &cobra.Command{
	Use:   "tickets",
	Short: "Ingresso.com CLI",
	Long:  `You can find the sessions, theaters and best seats all from the terminal :)`,
	Run: func(cmd *cobra.Command, args []string) {
		println(`
Ingresso.com CLI
You can find the sessions, theaters and best seats all from the terminal :)
use "help" to get all options
`)
	},
}

var byCity = &cobra.Command{
	Use:   "city",
	Short: "Find Today Events By City",
	Long:  `Find Events Playing Today in Your City`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Flag("city")
		GetEventsByCity()
	},
}

var byMovie = &cobra.Command{
	Use:   "event",
	Short: "Find Current Events Sessions",
	Long:  `Find Events Sessions in Your City`,
	Run: func(cmd *cobra.Command, args []string) {
		flag := cmd.Flag("city")
		println(flag.Value.String())
		cmd.Flag("date")
		cmd.Flag("type")

		GetSessionsByMovie()
	},
}

func Execute() {
	rootCmd.AddCommand(byCity, byMovie, versionCmd)
	byMovie.Flags().String("city", "", "insert your city name")
	byCity.Flags().String("city", "", "insert your city name")
	byMovie.Flags().String("date", "", "insert the session date")
	byMovie.Flags().String("type", "", "insert the session type ex: [legendado, dublado]")
	rootCmd.Execute()
	os.Exit(1)
}
