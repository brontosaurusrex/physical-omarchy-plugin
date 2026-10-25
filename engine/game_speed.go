package main

import "strconv"

const gameSpeedStorageKey = "physical.gameSpeedPercent"

var gameSpeedPercent = 100

func validGameSpeed(percent int) bool {
	switch percent {
	case 80, 90, 100, 150, 200:
		return true
	}
	return false
}

func loadGameSpeedPreference() {
	if percent, found := readStoredInt(gameSpeedStorageKey); found {
		previous := percent
		if percent == 110 {
			percent = 150
		} else if percent == 120 {
			percent = 200
		}
		if validGameSpeed(percent) {
			gameSpeedPercent = percent
			if percent != previous {
				writeStoredInt(gameSpeedStorageKey, percent)
			}
		}
	}
}

func setGameSpeed(percent int) {
	if !validGameSpeed(percent) {
		return
	}
	gameSpeedPercent = percent
	writeStoredInt(gameSpeedStorageKey, percent)
	if !levelAdvancePending && !gameOver {
		if levelRunStats.elapsedSeconds == 0 {
			levelRunStats.speedPercent = percent
			levelCurrentBests = loadLevelBestStats(currentLevelIndex)
			levelPreviousBests = levelCurrentBests
			if fullGameRun.active && currentLevelIndex == 0 && fullGameRun.elapsedSeconds == 0 {
				fullGameRun.speedPercent = percent
			}
		} else if levelRunStats.speedPercent != percent {
			levelRunStats.mixedSpeed = true
			levelCurrentBests = levelBestStatsData{}
			levelPreviousBests = levelBestStatsData{}
			markLevelRunAssisted()
			if fullGameRun.active {
				fullGameRun.mixedSpeed = true
			}
		}
	}
	// Start a fresh measurement window after a pace change; the physics
	// accumulator and interpolation history continue without a discontinuity.
	physicsSampleElapsed, physicsSampleComputeTime, physicsSampleDroppedTime = 0, 0, 0
	physicsSampleSteps = 0
	showStatus("@"+strconv.Itoa(percent)+"%", 2.0)
}

func gameSpeedShortcut(key string) bool {
	percent := 0
	switch key {
	case "6":
		percent = 80
	case "7":
		percent = 90
	case "8":
		percent = 100
	case "9":
		percent = 150
	case "0":
		percent = 200
	default:
		return false
	}
	setGameSpeed(percent)
	return true
}

func gameSpeedScale() float64 { return float64(gameSpeedPercent) / 100 }

func speedRecordKey(key string, percent int) string {
	if !validGameSpeed(percent) || percent == 100 {
		return key // Keep all existing 100% records at their original keys.
	}
	return key + ".at" + strconv.Itoa(percent)
}

func gameSpeedResultLabel(percent int, mixed bool) string {
	if mixed {
		return "MIXED SPEED"
	}
	return "@" + strconv.Itoa(percent) + "%"
}

func trackFullGameSpeed() {
	if fullGameRun.active && fullGameRun.speedPercent != gameSpeedPercent {
		fullGameRun.mixedSpeed = true
		fullGameRun.recordEligible = false
	}
}
