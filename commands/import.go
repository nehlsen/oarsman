package commands

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/olympum/oarsman/s4"
	"github.com/olympum/oarsman/util"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"os"
	"time"
)

var replay bool
var inputFile string

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import workout data from database",
	Long: `
Imports one or multiple workouts into the database
as RAW (40Hz JSON formatted feed).`,
	Run: func(cmd *cobra.Command, args []string) {
		InitializeConfig()

		if inputFile == "" {
			jww.ERROR.Println("Nothing to import")
			return
		}
		// Parse input file path to construct the fully qualified file name
		// Write output file using a UUID as file name
		eventChannel := make(chan s4.AtomicEvent)
		aggregateEventChannel := make(chan s4.AggregateEvent)
		collector := s4.NewEventCollector(aggregateEventChannel)
		go collector.Run()

		s, err := s4.NewReplayS4(eventChannel, aggregateEventChannel, replay, inputFile, replay)
		if err != nil {
			// TODO
			return
		}

		fqOfn := viper.GetString("TempFolder") + string(os.PathSeparator) + randomId()
		go s4.Logger(eventChannel, fqOfn)

		s.Run(nil)

		activity := collector.Activity
		jww.INFO.Printf("Parsed activity with start time %d\n", activity.StartTimeMilliseconds)

		database, error := WorkoutDatabase()
		if error != nil {
			// TODO
			return
		}
		defer database.Close()

		database.InsertActivity(activity) // move file to workout folder

		workoutFile := viper.GetString("WorkoutFolder") + string(os.PathSeparator) + util.MillisToZulu(activity.StartTimeMilliseconds) + ".log"
		os.Rename(fqOfn, workoutFile)
	},
}

func init() {
	importCmd.Flags().BoolVar(&replay, "replay", false, "print to stdout using precise time the original recorded the raw data packets")
	importCmd.Flags().StringVar(&inputFile, "input", "", "input file to import")
}

func randomId() string {
	size := 32
	rb := make([]byte, size)
	_, err := rand.Read(rb)

	if err != nil {
		jww.ERROR.Println(err)
		return string(time.Now().Nanosecond())
	}

	return base64.URLEncoding.EncodeToString(rb)
}
