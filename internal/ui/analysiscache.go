package ui

import (
	"path/filepath"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/analysiscache"
)

func (v *viewer) analysisRoots() similarity.CacheRoots {
	return similarity.CacheRoots{GeneralDir: v.analysisDir, FavoritesDir: v.favorites.Dir()}
}

func (v *viewer) searchCachePolicy() similarity.CachePolicy {
	return similarity.CachePolicy{Roots: v.analysisRoots(), FavoriteEnabled: v.explorer.Settings().CacheFavorites,
		LooseEnabled: v.settings.looseAnalysisCache, GeneralLimitBytes: uint64(v.settings.analysisCacheMiB) * 1024 * 1024}
}

func (v *viewer) registerAnalysisCache(prefs preferences.State) {
	v.settings.looseAnalysisCache, v.settings.analysisCacheMiB = prefs.SimilarityLooseCache, prefs.AnalysisCacheLimitMiB
	if v.settings.analysisCacheMiB <= 0 {
		v.settings.analysisCacheMiB = 2048
	}
	if root := v.app.Cache().RootURI(); root != nil && root.Scheme() == "file" {
		v.analysisDir = filepath.Join(root.Path(), "image-analysis")
	}
	v.analysisCache = analysiscache.New(analysisCacheHost{v}, analysiscache.Options{
		Roots: v.analysisRoots(), ConfirmClear: v.settingsWin.ConfirmClearAnalysis, Changed: v.syncMenus,
	})
	v.analysisCache.SetPolicy(v.settings.looseAnalysisCache, v.settings.analysisCacheMiB)
	v.settingsWin.SetCacheTab(func() fyne.CanvasObject {
		v.analysisCache.SetRoots(v.analysisRoots())
		return v.analysisCache.Content(v.settings.looseAnalysisCache, v.settings.analysisCacheMiB)
	}, v.analysisCache.Close)
}

type analysisCacheHost struct{ v *viewer }

func (h analysisCacheHost) Quiesce(reason analysiscache.QuiesceReason) []<-chan struct{} {
	v := h.v
	v.explorerInput.prepareOp.invalidate()
	v.explorerInput.prepare = nil
	barriers := []<-chan struct{}{v.explorer.Suspend()}
	if reason == analysiscache.AutomaticEviction {
		return append(barriers, v.visualsearch.CacheWritesRevoked())
	}
	return append(barriers, v.visualsearch.Suspend())
}

func (h analysisCacheHost) ApplyPolicy(enabled bool, limitMiB int) {
	v := h.v
	v.settings.looseAnalysisCache, v.settings.analysisCacheMiB = enabled, limitMiB
	v.visualsearch.SetCachePolicy(v.searchCachePolicy())
	preferences.Save(v.app, v.currentPreferences())
}

func (v *viewer) analysisMaintenanceBusy() bool {
	return v.analysisCache != nil && v.analysisCache.Busy()
}

func (v *viewer) favoriteSaved() {
	if v.visualsearch != nil {
		v.visualsearch.FavoriteSaved()
	}
}
