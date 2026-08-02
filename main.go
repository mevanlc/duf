package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	wildcard "github.com/IGLOU-EU/go-wildcard"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/muesli/termenv"
	flag "github.com/spf13/pflag"
	"golang.org/x/term"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	env   = termenv.EnvColorProfile()
	theme Theme

	groups        = []string{localDevice, networkDevice, fuseDevice, specialDevice, loopsDevice, bindsMount}
	allowedValues = strings.Join(groups, ", ")

	all         = flag.Bool("all", false, "include pseudo, duplicate, inaccessible file systems")
	hideDevices = flag.String("hide", "", "hide specific devices, separated with commas:\n"+allowedValues)
	hideFs      = flag.StringP("hide-fs", "F", "", "hide specific filesystems, separated with commas")
	hideMp      = flag.StringP("hide-mp", "U", "", "hide specific mount points, separated with commas (supports wildcards)")
	dereference = flag.BoolP("dereference", "L", false, "dereference symlinks matched by mount point filters")
	gte         = flag.String("gte", "", "only show entries with a total size greater than or equal to this size")
	onlyDevices = flag.StringP("only", "i", "", "show only specific devices, separated with commas:\n"+allowedValues)
	onlyFs      = flag.StringP("only-fs", "f", "", "only specific filesystems, separated with commas")
	onlyMp      = flag.StringP("only-mp", "u", "", "only specific mount points, separated with commas (supports wildcards)")

	output   = flag.StringP("output", "o", "", "output fields: "+strings.Join(columnIDs(), ", "))
	sortBy   = flag.StringP("sort", "R", "mountpoint", "sort output by: "+strings.Join(columnIDs(), ", "))
	width    = flag.Uint("width", 0, "max output width")
	themeOpt = flag.String("theme", defaultThemeName(), "color themes: dark, light, ansi")
	styleOpt = flag.String("style", defaultStyleName(), "style: unicode, ascii")

	availThreshold = flag.String("avail-threshold", "10G,1G", "specifies the coloring threshold (yellow, red) of the avail column, must be integer with optional SI prefixes")
	usageThreshold = flag.String("usage-threshold", "0.5,0.9", "specifies the coloring threshold (yellow, red) of the usage bars as a floating point number from 0 to 1")

	_          = flag.BoolP("human-readable", "h", false, "ignored, just for df compatibility")
	inodes     = flag.Bool("inodes", false, "list inode information instead of block usage")
	jsonOutput = flag.BoolP("json", "J", false, "output devices in JSON format")
	warns      = flag.Bool("warnings", false, "output all warnings to STDERR")
	version    = flag.Bool("version", false, "display version")
)

type duCompatibilityFlag struct {
	name          string
	shorthand     string
	requiresValue bool
}

var duNoopCompatibilityFlags = []duCompatibilityFlag{
	{name: "du-all", shorthand: "a"},
	{name: "du-apparent-size", shorthand: "A"},
	{name: "du-bytes", shorthand: "b"},
	{name: "du-block-size", shorthand: "B", requiresValue: true},
	{name: "du-total", shorthand: "c"},
	{name: "du-max-depth", shorthand: "d", requiresValue: true},
	{name: "du-gibibytes", shorthand: "g"},
	{name: "du-ignore", shorthand: "I", requiresValue: true},
	{name: "du-kibibytes", shorthand: "k"},
	{name: "du-count-links", shorthand: "l"},
	{name: "du-mebibytes", shorthand: "m"},
	{name: "du-ignore-nodump", shorthand: "n"},
	{name: "du-summarize", shorthand: "s"},
	{name: "du-separate-dirs", shorthand: "S"},
	{name: "du-one-file-system", shorthand: "x"},
	{name: "du-exclude-from", shorthand: "X", requiresValue: true},
}

func registerDUCompatibilityFlags(flagSet *flag.FlagSet, threshold *string, warnings, dereferenceSymlinks *bool) {
	for _, compatibilityFlag := range duNoopCompatibilityFlags {
		usage := "ignored, just for du compatibility"
		if compatibilityFlag.requiresValue {
			flagSet.StringP(compatibilityFlag.name, compatibilityFlag.shorthand, "", usage)
		} else {
			flagSet.BoolP(compatibilityFlag.name, compatibilityFlag.shorthand, false, usage)
		}
		_ = flagSet.MarkHidden(compatibilityFlag.name)
	}

	flagSet.StringVarP(threshold, "du-threshold", "t", "", "alias for --gte")
	_ = flagSet.MarkHidden("du-threshold")
	flagSet.BoolVarP(warnings, "du-report-errors", "r", false, "alias for --warnings")
	_ = flagSet.MarkHidden("du-report-errors")
	flagSet.BoolVarP(dereferenceSymlinks, "du-dereference-args", "D", false, "alias for --dereference")
	_ = flagSet.MarkHidden("du-dereference-args")
	flagSet.BoolVarP(dereferenceSymlinks, "du-dereference-command-line", "H", false, "alias for --dereference")
	_ = flagSet.MarkHidden("du-dereference-command-line")
	flagSet.BoolFuncP("du-no-dereference", "P", "disable --dereference", func(string) error {
		*dereferenceSymlinks = false
		return nil
	})
	_ = flagSet.MarkHidden("du-no-dereference")
}

func init() {
	registerDUCompatibilityFlags(flag.CommandLine, gte, warns, dereference)
}

// renderJSON encodes the JSON output and prints it.
func renderJSON(m []Mount) error {
	output, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return fmt.Errorf("error formatting the json output: %s", err)
	}

	fmt.Println(string(output))
	return nil
}

// parseColumns parses the supplied output flag into a slice of column indices.
func parseColumns(cols string) ([]int, error) {
	var i []int

	s := strings.Split(cols, ",")
	for _, v := range s {
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			continue
		}

		col, err := stringToColumn(v)
		if err != nil {
			return nil, err
		}

		i = append(i, col)
	}

	return i, nil
}

// parseStyle converts user-provided style option into a table.Style.
func parseStyle(styleOpt string) (table.Style, error) {
	switch styleOpt {
	case "unicode":
		return table.StyleRounded, nil
	case "ascii":
		return table.StyleDefault, nil
	default:
		return table.Style{}, fmt.Errorf("unknown style option: %s", styleOpt)
	}
}

// parseCommaSeparatedValues parses comma separated string into a map.
func parseCommaSeparatedValues(values string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, v := range strings.Split(values, ",") {
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			continue
		}

		m[v] = struct{}{}
	}
	return m
}

// parseCaseInsensitiveCommaSeparatedValues parses comma separated strings and
// normalizes them to lowercase.
func parseCaseInsensitiveCommaSeparatedValues(values string) map[string]struct{} {
	return parseCommaSeparatedValues(strings.ToLower(values))
}

// validateGroups validates the parsed group maps.
func validateGroups(m map[string]struct{}) error {
	for k := range m {
		found := false
		for _, g := range groups {
			if g == k {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unknown device group: %s", k)
		}
	}

	return nil
}

// findInKey parse a slice of pattern to match the given key.
func findInKey(str string, km map[string]struct{}) bool {
	for p := range km {
		if wildcard.Match(p, str) {
			return true
		}
	}

	return false
}

func printVersion() {
	info, ok := debug.ReadBuildInfo()
	var buildTime time.Time
	var modified bool
	if ok {
		if len(Version) == 0 {
			vs := strings.Split(info.Main.Version, "-")
			if len(vs) >= 1 {
				Version = vs[0]
			}
		}

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(CommitSHA) == 0 {
					CommitSHA = setting.Value
					if len(CommitSHA) > 12 {
						CommitSHA = CommitSHA[:12]
					}
				}
			case "vcs.time":
				buildTime, _ = time.Parse(time.RFC3339, setting.Value)
			case "vcs.modified":
				modified, _ = strconv.ParseBool(setting.Value)
			}
		}
	}

	if Version == "" || Version == "(devel)" {
		Version = "(built from source)"
	}

	fmt.Printf("duf %s", Version)
	if len(CommitSHA) > 0 {
		if modified {
			CommitSHA += "+modified"
		}
		fmt.Printf(" (%s)", CommitSHA)
	}
	if !buildTime.IsZero() {
		fmt.Printf(" (built on %s)", buildTime.Format("2006-01-02"))
	}

	fmt.Println()
}

func main() {
	// hide -h from help, it's just for df compatibility
	_ = flag.CommandLine.MarkHidden("human-readable")
	flag.Parse()

	if *version {
		printVersion()
		os.Exit(0)
	}

	minimumTotalSize, err := parseMinimumTotalSize(*gte)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error parsing gte:", err)
		os.Exit(1)
	}

	// read mount table
	m, warnings, err := mounts()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	m = filterMountsByTotalSize(m, minimumTotalSize)

	// print JSON
	if *jsonOutput {
		if err = renderJSON(m); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return
	}

	// validate theme
	theme, err = loadTheme(*themeOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if env == termenv.ANSI {
		// enforce ANSI theme for limited color support
		theme, err = loadTheme("ansi")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	// validate style
	style, err := parseStyle(*styleOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// validate output columns
	columns, err := parseColumns(*output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(columns) == 0 {
		// no columns supplied, use defaults
		if *inodes {
			columns = []int{1, 6, 7, 8, 9, 10, 11}
		} else {
			columns = []int{1, 2, 3, 4, 5, 10, 11}
		}
	}

	// validate sort column
	sortCol, err := stringToSortIndex(*sortBy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// validate filters
	filters := FilterOptions{
		HiddenDevices:     parseCaseInsensitiveCommaSeparatedValues(*hideDevices),
		OnlyDevices:       parseCaseInsensitiveCommaSeparatedValues(*onlyDevices),
		HiddenFilesystems: parseCaseInsensitiveCommaSeparatedValues(*hideFs),
		OnlyFilesystems:   parseCaseInsensitiveCommaSeparatedValues(*onlyFs),
		HiddenMountPoints: parseCommaSeparatedValues(*hideMp),
		OnlyMountPoints:   parseCommaSeparatedValues(*onlyMp),
	}
	if *dereference {
		filters.HiddenMountPoints, err = dereferenceMountPointPatterns(m, filters.HiddenMountPoints)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		filters.OnlyMountPoints, err = dereferenceMountPointPatterns(m, filters.OnlyMountPoints)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	err = validateGroups(filters.HiddenDevices)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = validateGroups(filters.OnlyDevices)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// validate arguments
	if len(flag.Args()) > 0 {
		var mounts []Mount
		vis := map[string]struct{}{}

		for _, v := range flag.Args() {
			var fm []Mount
			fm, err = findMounts(m, v)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			// de-duplicate
			for _, v := range fm {
				if _, ok := vis[v.Mountpoint]; !ok {
					mounts = append(mounts, v)
					vis[v.Mountpoint] = struct{}{}
				}
			}
		}

		m = mounts
	}

	// validate availability thresholds
	availbilityThresholds := strings.Split(*availThreshold, ",")
	if len(availbilityThresholds) != 2 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("error parsing avail-threshold: invalid option '%s'", *availThreshold))
		os.Exit(1)
	}
	for _, threshold := range availbilityThresholds {
		_, err = stringToSize(threshold)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing avail-threshold:", err)
			os.Exit(1)
		}
	}

	// validate usage thresholds
	usageThresholds := strings.Split(*usageThreshold, ",")
	if len(usageThresholds) != 2 {
		fmt.Fprintln(os.Stderr, fmt.Errorf("error parsing usage-threshold: invalid option '%s'", *usageThreshold))
		os.Exit(1)
	}
	for _, threshold := range usageThresholds {
		_, err = strconv.ParseFloat(threshold, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing usage-threshold:", err)
			os.Exit(1)
		}
	}

	// print out warnings
	if *warns {
		for _, warning := range warnings {
			fmt.Fprintln(os.Stderr, warning)
		}
	}

	// detect terminal width
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	if isTerminal && *width == 0 {
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			*width = uint(w)
		}
	}
	if *width == 0 {
		*width = 80
	}

	// print tables
	renderTables(m, filters, TableOptions{
		Columns:   columns,
		SortBy:    sortCol,
		Style:     style,
		StyleName: *styleOpt,
	})
}
