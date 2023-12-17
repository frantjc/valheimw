package valheim

import (
	"fmt"
	"path/filepath"
	"time"
)

type Preset string

var (
	PresetNormal    Preset = "normal"
	PresetCasual    Preset = "casual"
	PresetEasy      Preset = "easy"
	PresetHard      Preset = "hard"
	PresetHardcore  Preset = "hardcore"
	PresetImmersive Preset = "immersive"
	PresetHammer    Preset = "hammer"
)

type CombatModifier string

var (
	CombatModifierVeryEasy CombatModifier = "veryeasy"
	CombatModifierEasy     CombatModifier = "easy"
	CombatModifierHard     CombatModifier = "hard"
	CombatModifierVeryHard CombatModifier = "veryhard"
)

type DeathPenaltyModifier string

var (
	DeathPenaltyModifierCasual   DeathPenaltyModifier = "casual"
	DeathPenaltyModifierVeryEasy DeathPenaltyModifier = "veryeasy"
	DeathPenaltyModifierEasy     DeathPenaltyModifier = "easy"
	DeathPenaltyModifierHard     DeathPenaltyModifier = "hard"
	DeathPenaltyModifierHardcore DeathPenaltyModifier = "hardcore"
)

type ResourceModifier string

var (
	ResourceModifierMuchLess DeathPenaltyModifier = "muchless"
	ResourceModifierLess     DeathPenaltyModifier = "less"
	ResourceModifierMore     DeathPenaltyModifier = "more"
	ResourceModifierMuchMore DeathPenaltyModifier = "muchmore"
	ResourceModifierMost     DeathPenaltyModifier = "most"
)

type RaidModifier string

var (
	RaidModifierNone     RaidModifier = "none"
	RaidModifierMuchLess RaidModifier = "muchless"
	RaidModifierLess     RaidModifier = "less"
	RaidModifierMore     RaidModifier = "more"
	RaidModifierMuchMore RaidModifier = "muchmore"
)

type PortalModifier string

var (
	PortalModifierCasual   PortalModifier = "casual"
	PortalModifierHard     PortalModifier = "hard"
	PortalModifierVeryHard PortalModifier = "veryhard"
)

// Opts is a helper struct to build arguments
// to pass to the Valheim executable.
type Opts struct {
	Name     string
	Port     int64
	World    string
	Password string
	SaveDir  string

	Public bool

	LogFile string

	SaveInterval time.Duration
	Backups      int64
	BackupShort  time.Duration
	BackupLong   time.Duration

	Crossplay bool

	InstanceID string

	Preset

	CombatModifier
	DeathPenaltyModifier
	ResourceModifier
	RaidModifier
	PortalModifier

	NoBuildCost  bool
	PlayerEvents bool
	PassiveMobs  bool
	NoMap        bool
}

// ToArgs transforms Opts into an array
// of strings to pass to the Valheim Executable.
func (o *Opts) ToArgs() []string {
	args := []string{}

	if o.Name != "" {
		args = append(args, "-name", o.Name)
	}

	if o.Port != 0 {
		args = append(args, "-port", fmt.Sprint(o.Port))
	}

	if o.World != "" {
		args = append(args, "-world", o.World)
	}

	if o.Password != "" {
		args = append(args, "-password", o.Password)
	}

	if o.SaveDir != "" {
		args = append(args, "-savedir", o.SaveDir)
	}

	if !o.Public {
		args = append(args, "-public", "0")
	}

	if o.LogFile != "" {
		args = append(args, "-logFile", filepath.Clean(o.LogFile))
	}

	if o.SaveInterval != 0 {
		args = append(args, "-saveinterval", fmt.Sprint(int64(o.SaveInterval.Seconds())))
	}

	if o.Backups != 0 {
		args = append(args, "-backups", fmt.Sprint(o.Backups))
	}

	if o.BackupShort != 0 {
		args = append(args, "-backupshort", fmt.Sprint(int64(o.BackupShort.Seconds())))
	}

	if o.SaveInterval != 0 {
		args = append(args, "-backuplong", fmt.Sprint(int64(o.BackupLong.Seconds())))
	}

	if o.Crossplay {
		args = append(args, "-crossplay")
	}

	if o.InstanceID != "" {
		args = append(args, "-instanceid", o.InstanceID)
	}

	if o.Preset != "" {
		args = append(args, "-preset", string(o.Preset))
	}

	if o.CombatModifier != "" {
		args = append(args, "-modifier", "combat", string(o.CombatModifier))
	}

	if o.DeathPenaltyModifier != "" {
		args = append(args, "-modifier", "deathpenalty", string(o.DeathPenaltyModifier))
	}

	if o.ResourceModifier != "" {
		args = append(args, "-modifier", "resources", string(o.ResourceModifier))
	}

	if o.RaidModifier != "" {
		args = append(args, "-modifier", "raids", string(o.RaidModifier))
	}

	if o.PortalModifier != "" {
		args = append(args, "-modifier", "portals", string(o.PortalModifier))
	}

	if o.NoBuildCost {
		args = append(args, "-setkey", "nobuildcost")
	}

	if o.PlayerEvents {
		args = append(args, "-setkey", "playerevents")
	}

	if o.PassiveMobs {
		args = append(args, "-setkey", "passivemobs")
	}

	if o.NoMap {
		args = append(args, "-setkey", "nomap")
	}

	return args
}
