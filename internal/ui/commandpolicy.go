package ui

import "github.com/frathe/picfetch/internal/filesort"

// Admission observes values only. Payload capture, feature state and effects
// remain with the command's existing execution owner.
type commandID uint8

const (
	commandSave commandID = iota
	commandCopy
	commandCopyImage
	commandCopyFiles
	commandCopyPath
	commandCopyRegion
	commandSelectAll
	commandOpen
	commandOpenChooser
	commandCloseFiles
	commandRestore
	commandFavoriteOpen
	commandFavoriteAdd
	commandFavoriteManage
	commandExport
	commandTrash
	commandReveal
	commandWallpaper
	commandViewer
	commandGrid
	commandPictureFrame
	commandCompare
	commandExif
	commandSettings
	commandHelp
	commandSpiral
	commandExplorer
	commandLocationMap
	commandMosaic
	commandSort
	commandHideDuplicates
	commandBrowseDuplicates
	commandSearch
	commandNavigate
	commandSelectImage
	commandRotate
	commandReset
	commandZoom
	commandMerge
	commandInfo
	commandShuffle
	commandInterval
)

type commandIntent uint8

const (
	intentAction commandIntent = iota
	intentShow
	intentToggle
	intentEditing
	intentContinuation
)

type commandRoute uint8

const (
	_ commandRoute = iota // Zero-value requests use the direct route.
	routeMenu
	routeShortcut
	routeKey
	routeDelivery
)

type commandRequest struct {
	command  commandID
	intent   commandIntent
	route    commandRoute
	sortMode filesort.Mode
}

type commandSurface uint8

const (
	surfaceViewer commandSurface = iota
	surfaceGrid
	surfacePictureFrame
	surfaceComparison
	surfaceExplorer
	surfaceLocationMap
)

type commandInput uint8

const (
	_ commandInput = iota // Zero-value contexts have ordinary viewer input.
	inputMenu
	inputEditor
	inputModal
)

type commandContext struct {
	surface                                               commandSurface
	input                                                 commandInput
	cohortVisit, locationVisit                            bool
	searchVisit, variantsVisit                            bool
	regionActive, regionBusy                              bool
	clipboardBusy, stopping, canSave                      bool
	editorFocused                                         bool
	clipboardClosed                                       bool
	hasFiles, hasImage, loading                           bool
	hasCollection                                         bool
	hasPixels                                             bool
	gridTargets                                           bool
	fileWorkActive, chooserClosed, hasSession             bool
	pendingLaunchFrame                                    bool
	canExport, canWallpaper                               bool
	canCompare, displayed, exifOpen, manualOpen           bool
	analysisBusy, explorerCanRetry, canMosaic, mosaicOpen bool
	hideDuplicates, browsingDuplicates, hasSearchTarget   bool
	variantGroupSize                                      int
	sortMode                                              filesort.Mode
	canNavigate                                           bool
}

type commandRefusal uint8

const (
	_ commandRefusal = iota // An allowed decision's zero value has no refusal.
	refusalUnavailable
	refusalModal
	refusalSurface
	refusalRegionBusy
	refusalClipboardBusy
	refusalComparisonOpen
	refusalCompareTargets
)

type commandTarget uint8

const (
	targetNone commandTarget = iota
	targetDisplayedImage
	targetEditor
	targetRegion
	targetGridFiles
	targetCurrentFile
	targetCollection
)

type commandDecision struct {
	allowed     bool
	refusal     commandRefusal
	yieldRegion bool
	target      commandTarget
}

func decideCommand(request commandRequest, context commandContext) commandDecision {
	if context.stopping {
		return commandDecision{refusal: refusalUnavailable}
	}
	if request.intent == intentEditing && context.editorFocused && (request.command == commandCopy || request.command == commandSelectAll) {
		return commandDecision{allowed: true, target: targetEditor}
	}
	ownedLoad := request.command == commandSelectImage && request.intent == intentContinuation
	if context.input == inputModal && !ownedLoad {
		return commandDecision{refusal: refusalModal}
	}
	if context.surface == surfaceComparison && request.command != commandHelp {
		if request.command == commandOpen || request.command == commandOpenChooser {
			return commandDecision{refusal: refusalComparisonOpen}
		}
		return commandDecision{refusal: refusalSurface}
	}
	// Grid closes before delivering its selected cohort image. The covered
	// Explorer surface is briefly visible until ShowImage hides it.
	cohortSelection := request.command == commandSelectImage && context.cohortVisit
	if !ownedLoad && !cohortSelection && (context.surface == surfaceExplorer || context.surface == surfaceLocationMap) {
		switch request.command {
		case commandOpen, commandOpenChooser, commandCloseFiles, commandRestore, commandFavoriteOpen, commandFavoriteAdd, commandFavoriteManage, commandViewer, commandSettings, commandHelp, commandSpiral, commandExplorer, commandLocationMap, commandMosaic:
		default:
			return commandDecision{refusal: refusalSurface}
		}
	}
	keepsRegion := request.command == commandCopy || request.command == commandCopyRegion || request.command == commandZoom
	if context.regionBusy && !keepsRegion {
		return commandDecision{refusal: refusalRegionBusy}
	}
	allowed := func(target commandTarget) commandDecision {
		return commandDecision{allowed: true, yieldRegion: context.regionActive && !keepsRegion, target: target}
	}
	switch request.command {
	case commandNavigate:
		if context.canNavigate {
			return allowed(targetCollection)
		}
	case commandSelectImage:
		if context.hasFiles {
			return allowed(targetCurrentFile)
		}
	case commandRotate, commandReset, commandZoom:
		if context.hasImage && context.surface != surfaceGrid {
			return allowed(targetDisplayedImage)
		}
	case commandMerge, commandShuffle:
		return allowed(targetCollection)
	case commandInfo:
		if context.surface != surfaceGrid {
			return allowed(targetDisplayedImage)
		}
	case commandInterval:
		if context.surface == surfacePictureFrame {
			return allowed(targetCollection)
		}
	case commandSort:
		if request.route == routeKey && !context.canNavigate {
			break
		}
		if request.intent == intentToggle || request.sortMode != context.sortMode {
			return allowed(targetCollection)
		}
	case commandHideDuplicates:
		// D pre-arms the standing preference even before the first drop;
		// the explicit menu action has always required a loaded collection.
		if (context.hasFiles || request.route != routeMenu) && !context.variantsVisit && !context.cohortVisit && !context.locationVisit && !context.searchVisit {
			return allowed(targetCollection)
		}
	case commandBrowseDuplicates:
		if context.hasFiles && context.surface != surfacePictureFrame && !context.cohortVisit && !context.locationVisit && !context.searchVisit {
			if request.intent != intentShow || context.browsingDuplicates || context.hideDuplicates && context.variantGroupSize >= 2 {
				return allowed(targetCollection)
			}
		}
	case commandSearch:
		if context.hasSearchTarget && !context.analysisBusy && !context.fileWorkActive && !context.locationVisit && context.surface != surfacePictureFrame && !context.editorFocused {
			return allowed(targetCurrentFile)
		}
	case commandExplorer:
		if context.hasFiles && !context.fileWorkActive && !context.analysisBusy && (request.route == routeDelivery || context.surface != surfaceExplorer || context.explorerCanRetry) {
			return allowed(targetCollection)
		}
	case commandLocationMap:
		if !context.fileWorkActive {
			return allowed(targetCollection)
		}
	case commandMosaic:
		if context.mosaicOpen || context.canMosaic {
			return allowed(targetCollection)
		}
	case commandSettings, commandHelp, commandSpiral:
		if request.command == commandHelp && request.route == routeMenu && request.intent == intentShow && context.manualOpen {
			break
		}
		return allowed(targetNone)
	case commandViewer:
		if context.surface == surfaceGrid || context.surface == surfacePictureFrame || context.surface == surfaceExplorer || context.surface == surfaceLocationMap || context.locationVisit || context.cohortVisit || context.searchVisit {
			return allowed(targetCollection)
		}
	case commandGrid:
		if context.hasFiles && context.surface != surfacePictureFrame {
			if request.route == routeMenu && context.surface == surfaceGrid {
				break
			}
			return allowed(targetCollection)
		}
	case commandPictureFrame:
		if context.hasFiles && !context.variantsVisit && !context.cohortVisit && !context.locationVisit {
			if request.intent == intentShow && context.surface == surfacePictureFrame {
				break
			}
			return allowed(targetCollection)
		}
	case commandExif:
		if context.displayed && (request.route != routeMenu || !context.exifOpen) {
			return allowed(targetCurrentFile)
		}
	case commandCompare:
		if context.canCompare {
			return allowed(targetGridFiles)
		}
		return commandDecision{refusal: refusalCompareTargets}
	case commandExport:
		if context.canExport {
			return allowed(targetDisplayedImage)
		}
	case commandWallpaper:
		if context.canWallpaper {
			return allowed(targetDisplayedImage)
		}
	case commandReveal:
		if context.hasFiles {
			return allowed(targetCurrentFile)
		}
	case commandTrash:
		if context.surface == surfaceGrid {
			if context.gridTargets {
				return allowed(targetGridFiles)
			}
		} else if context.hasFiles {
			return allowed(targetCurrentFile)
		}
	case commandOpen, commandFavoriteOpen, commandFavoriteManage:
		return allowed(targetCollection)
	case commandOpenChooser:
		if !context.chooserClosed {
			return allowed(targetCollection)
		}
	case commandCloseFiles:
		// Before startup has submitted its first scan, a direct reset still
		// consumes --slideshow. Keep the empty File-menu item disabled.
		if context.hasCollection || context.fileWorkActive || request.route != routeMenu && context.pendingLaunchFrame {
			return allowed(targetCollection)
		}
	case commandRestore:
		if context.hasSession {
			return allowed(targetCollection)
		}
	case commandFavoriteAdd:
		if context.hasFiles {
			return allowed(targetCollection)
		}
	case commandSave:
		if context.canSave {
			return commandDecision{allowed: true, yieldRegion: context.regionActive, target: targetDisplayedImage}
		}
	case commandSelectAll:
		if context.surface == surfaceGrid {
			return allowed(targetGridFiles)
		}
	case commandCopyRegion:
		if context.regionActive || context.hasImage && !context.loading && context.surface == surfaceViewer {
			return allowed(targetRegion)
		}
	case commandCopy, commandCopyImage, commandCopyFiles, commandCopyPath:
		if context.clipboardClosed {
			break
		}
		if request.command == commandCopy && context.regionActive {
			return allowed(targetRegion)
		}
		if context.clipboardBusy {
			return commandDecision{refusal: refusalClipboardBusy}
		}
		// The enclosing case admits only these four clipboard commands.
		//goland:noinspection GoSwitchMissingCasesForIotaConsts
		switch request.command {
		case commandCopyPath:
			if context.hasFiles {
				return allowed(targetCurrentFile)
			}
		case commandCopyFiles:
			if context.surface == surfaceGrid && context.gridTargets {
				return allowed(targetGridFiles)
			}
		case commandCopy:
			if context.surface == surfaceGrid {
				if context.gridTargets {
					return allowed(targetGridFiles)
				}
				break
			}
			if context.hasPixels {
				return allowed(targetDisplayedImage)
			}
		case commandCopyImage:
			if context.hasPixels {
				return allowed(targetDisplayedImage)
			}
		}
	}
	return commandDecision{refusal: refusalUnavailable}
}
