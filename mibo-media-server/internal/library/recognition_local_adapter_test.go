package library

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/atlan/mibo-media-server/internal/database"
	"github.com/atlan/mibo-media-server/internal/recognition"
	"github.com/atlan/mibo-media-server/internal/scanrecognition"
)

func TestBuildNewRecognitionSchemeOutputUsesScanRecognitionMovieFolder(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Movies/Inception (2010)/Inception.2010.1080p.BluRay.x265.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Inception", Year: intPtrForReduction(2010)})
	assertHasCandidate(t, output.Candidates, movieKey, recognition.CandidateTypeWork, recognition.WorkKindMovie)
	assertHasResourceParent(t, output.Candidates, movieKey, files[0].ID)
	assertEvidenceSource(t, output.Evidence, "scanrecognition")
}

func TestBuildNewRecognitionSchemeOutputUsesScanRecognitionCombinedSeasonFolder(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/电视剧/六尺之下 第一季[全13集][中文字幕].Six.Feet.Under.2001.1080p.WEB-DL.x265.AC3-BitsTV/Six.Feet.Under.S01E01.2001.1080p.WEB-DL.x265.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "local", StoragePath: "/library/电视剧/六尺之下 第一季[全13集][中文字幕].Six.Feet.Under.2001.1080p.WEB-DL.x265.AC3-BitsTV/Six.Feet.Under.S01E02.2001.1080p.WEB-DL.x265.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	seriesKey := recognition.SeriesWorkKey("Six Feet Under")
	seasonKey := recognition.SeasonWorkKey("Six Feet Under", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Six Feet Under", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Six Feet Under", SeasonNumber: 1, EpisodeNumber: 2})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, episodeOneKey, files[0].ID)
	assertHasResourceParent(t, output.Candidates, episodeTwoKey, files[1].ID)
	assertEvidenceSource(t, output.Evidence, "scanrecognition")
}

func TestBuildNewRecognitionSchemeOutputUsesRootMovieFile(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Movie A.2024.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Movie A", Year: intPtrForReduction(2024)})
	assertHasCandidate(t, output.Candidates, movieKey, recognition.CandidateTypeWork, recognition.WorkKindMovie)
	assertHasResourceParent(t, output.Candidates, movieKey, files[0].ID)
}

func TestBuildNewRecognitionSchemeOutputPrefersMovieFolderNameOverGenericFilename(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Movies/The Matrix (1999)/file.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "The Matrix", Year: intPtrForReduction(1999)})
	assertHasCandidate(t, output.Candidates, movieKey, recognition.CandidateTypeWork, recognition.WorkKindMovie)
	assertHasResourceParent(t, output.Candidates, movieKey, files[0].ID)
}

func TestMovieDisplayTitleUsesFolderPrimaryCandidate(t *testing.T) {
	model := extractFilenameSignalModel("/library/电影/辣身舞[中文字幕].Dirty.Dancing.1987.1080p/Dirty.Dancing.1987.1080p.mkv")
	node := &scanrecognition.DirectoryNode{Path: "/library/电影/辣身舞[中文字幕].Dirty.Dancing.1987.1080p", Name: "辣身舞[中文字幕].Dirty.Dancing.1987.1080p", Kind: scanrecognition.DirectoryKindMovie}
	got := movieDisplayTitle("", model, node, scanrecognition.DirectoryKindMovie, "Dirty Dancing")
	if got != "辣身舞" {
		t.Fatalf("expected folder primary display title, got %q", got)
	}
}

func TestMaterializedMovieMetadataUsesFolderDisplayTitleWithEnglishWorkKey(t *testing.T) {
	files := []database.InventoryFile{{ID: 1, StorageProvider: "local", StoragePath: "/library/电影/Dirty.Dancing.1987/Dirty.Dancing.1987.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"}}
	output := buildScanRecognitionManifestOutput(files, "/library", nil)
	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Dirty Dancing", Year: intPtrForReduction(1987)})
	for _, candidate := range output.Candidates {
		if candidate.CandidateKey != movieKey || candidate.CandidateType != recognition.CandidateTypeWork || candidate.CandidateRole != recognition.WorkKindMovie {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(candidate.EvidenceJSON), &payload); err != nil {
			t.Fatalf("unmarshal evidence: %v", err)
		}
		if payload["work_title"] != "Dirty Dancing" {
			t.Fatalf("expected work_title Dirty Dancing, got %#v", payload)
		}
		return
	}
	t.Fatalf("expected movie candidate %q in %#v", movieKey, output.Candidates)
}

func TestMovieDisplayTitleUsesMovieDirectoryWhenDirectoryIsScanRoot(t *testing.T) {
	rootPath := "/电影/合集5-2/辣身舞[中文字幕].Dirty.Dancing.1987.1080p.BluRay.DTS.x265-10bit-CHDBits"
	files := []database.InventoryFile{{ID: 1, StorageProvider: "openlist", StoragePath: rootPath + "/Dirty.Dancing.1987.1080p.BluRay.DTS.x265-10bit-CHDBits.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"}}
	output := buildScanRecognitionManifestOutput(files, rootPath, nil)
	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Dirty Dancing", Year: intPtrForReduction(1987)})
	for _, candidate := range output.Candidates {
		if candidate.CandidateKey != movieKey || candidate.CandidateType != recognition.CandidateTypeWork || candidate.CandidateRole != recognition.WorkKindMovie {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(candidate.EvidenceJSON), &payload); err != nil {
			t.Fatalf("unmarshal evidence: %v", err)
		}
		if payload["title"] != "辣身舞" || payload["work_title"] != "Dirty Dancing" {
			t.Fatalf("expected folder display title and English work title, got %#v", payload)
		}
		return
	}
	t.Fatalf("expected movie candidate %q in %#v", movieKey, output.Candidates)
}

func TestBuildNewRecognitionSchemeOutputUsesSplitSeasonNumericEpisodes(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Show/Season 1/01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "local", StoragePath: "/library/Show/Season 1/02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	seriesKey := recognition.SeriesWorkKey("Show")
	seasonKey := recognition.SeasonWorkKey("Show", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Show", SeasonNumber: 1, EpisodeNumber: 2})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, episodeOneKey, files[0].ID)
	assertHasResourceParent(t, output.Candidates, episodeTwoKey, files[1].ID)
}

func TestBuildNewRecognitionSchemeOutputPrefersSeriesFolderNameOverEpisodeFilename(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Shows/The Last of Us/Season 1/file.S01E01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "local", StoragePath: "/library/Shows/The Last of Us/Season 1/file.S01E02.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	seriesKey := recognition.SeriesWorkKey("The Last of Us")
	seasonKey := recognition.SeasonWorkKey("The Last of Us", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "The Last of Us", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "The Last of Us", SeasonNumber: 1, EpisodeNumber: 2})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertNoMovieWorkForTitle(t, output.Candidates, "The Last of Us")
}

func TestBuildScanRecognitionManifestOutputUsesEpisodeOnlyFolderAsSeries(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E01.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "local", StoragePath: "/library/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E02.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, StorageProvider: "local", StoragePath: "/library/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.SP.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", nil)

	seriesKey := recognition.SeriesWorkKey("Anata no Ban Desu")
	seasonKey := recognition.SeasonWorkKey("Anata no Ban Desu", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 2})
	specialKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 21})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, episodeOneKey, files[0].ID)
	assertHasResourceParent(t, output.Candidates, episodeTwoKey, files[1].ID)
	assertHasResourceParent(t, output.Candidates, specialKey, files[2].ID)
}

func TestBuildScanRecognitionManifestOutputUsesEpisodeOnlyFolderAsSeriesAtRoot(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "openlist", StoragePath: "/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E01.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "openlist", StoragePath: "/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.E02.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, StorageProvider: "openlist", StoragePath: "/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV/Anata.no.Ban.Desu.SP.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/", nil)

	seriesKey := recognition.SeriesWorkKey("Anata no Ban Desu")
	seasonKey := recognition.SeasonWorkKey("Anata no Ban Desu", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 2})
	specialKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 21})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, episodeOneKey, files[0].ID)
	assertHasResourceParent(t, output.Candidates, episodeTwoKey, files[1].ID)
	assertHasResourceParent(t, output.Candidates, specialKey, files[2].ID)
	assertNoMovieWorkForTitle(t, output.Candidates, "Anata no Ban Desu")
	assertEvidenceSource(t, output.Evidence, "scanrecognition")
}

func TestBuildScanRecognitionManifestOutputUsesEpisodeOnlyLibraryRootAsSeries(t *testing.T) {
	rootPath := "/电视剧/轮到你了[全20集][中文字幕].Anata.no.Ban.Desu.E01-E20+SP.2019.1080p.WEB-DL.x265.AC3-BitsTV"
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "openlist", StoragePath: rootPath + "/Anata.no.Ban.Desu.E01.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "openlist", StoragePath: rootPath + "/Anata.no.Ban.Desu.E02.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, StorageProvider: "openlist", StoragePath: rootPath + "/Anata.no.Ban.Desu.SP.2019.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, rootPath, nil)

	seriesKey := recognition.SeriesWorkKey("Anata no Ban Desu")
	seasonKey := recognition.SeasonWorkKey("Anata no Ban Desu", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 2})
	specialKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Anata no Ban Desu", SeasonNumber: 1, EpisodeNumber: 21})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, episodeOneKey, files[0].ID)
	assertHasResourceParent(t, output.Candidates, episodeTwoKey, files[1].ID)
	assertHasResourceParent(t, output.Candidates, specialKey, files[2].ID)
	assertNoMovieWorkForTitle(t, output.Candidates, "Anata no Ban Desu")
	assertEvidenceSource(t, output.Evidence, "scanrecognition")
}

func TestBuildScanRecognitionManifestOutputAssignsIndexedAndResidualSpecialsInEpisodeGroup(t *testing.T) {
	rootPath := "/library/电视剧/窥探[全20集][中文字幕].Mouse.E01-E20+SP.2021.1080p.WEB-DL.x265.AC3-BitsTV"
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "openlist", StoragePath: rootPath + "/Mouse.E01.1080p.2021.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "openlist", StoragePath: rootPath + "/Mouse.E00.Restart.Highlight.Clips.1080p.2021.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, StorageProvider: "openlist", StoragePath: rootPath + "/Mouse.SP01.The.Predator.1080p.2021.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 4, StorageProvider: "openlist", StoragePath: rootPath + "/Mouse.SP02.The.Predator.1080p.2021.1080p.WEB-DL.x265.10bit.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library/电视剧", nil)

	seriesKey := recognition.SeriesWorkKey("Mouse")
	seasonKey := recognition.SeasonWorkKey("Mouse", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Mouse", SeasonNumber: 1, EpisodeNumber: 1})
	specialOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Mouse", SeasonNumber: 1, EpisodeNumber: 21})
	specialTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Mouse", SeasonNumber: 1, EpisodeNumber: 22})
	specialThreeKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Mouse", SeasonNumber: 1, EpisodeNumber: 23})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialThreeKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, specialThreeKey, files[1].ID)
	assertHasResourceParent(t, output.Candidates, specialOneKey, files[2].ID)
	assertHasResourceParent(t, output.Candidates, specialTwoKey, files[3].ID)
	assertNoMovieWorkForTitle(t, output.Candidates, "Mouse")
}

func TestBuildScanRecognitionManifestOutputTreatsResidualMovieLikeFileInEpisodeGroupAsSpecial(t *testing.T) {
	rootPath := "/library/电视剧/非关正义[全11集][简体字幕].Unfair.E01-E12+SP.2006.1080p.WEB-DL.x265.AC3-BitsTV"
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "openlist", StoragePath: rootPath + "/Unfair.E01.2006.1080p.WEB-DL.x265.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "openlist", StoragePath: rootPath + "/Unfair.E02.2006.1080p.WEB-DL.x265.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 3, StorageProvider: "openlist", StoragePath: rootPath + "/非关正义：完结篇.Unfair.the.End.2015.1080p.BluRay.x265.AC3-BitsTV.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library/电视剧", nil)

	seriesKey := recognition.SeriesWorkKey("Unfair")
	seasonKey := recognition.SeasonWorkKey("Unfair", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Unfair", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Unfair", SeasonNumber: 1, EpisodeNumber: 2})
	specialKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Unfair", SeasonNumber: 1, EpisodeNumber: 12})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, specialKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasResourceParent(t, output.Candidates, specialKey, files[2].ID)
	assertNoMovieWorkForTitle(t, output.Candidates, "Unfair")
}

func TestBuildScanRecognitionManifestOutputUsesEpisodeOnlyFolderWithHDTVEpisodesAsSeries(t *testing.T) {
	rootPath := "/library/电视剧/940920[全10集][粤语配音+中文字幕].ViuTV.940920.Complete.HDTV.1080i.H264-EntTV"
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "openlist", StoragePath: rootPath + "/ViuTV.940920.E01.HDTV.1080i.H264-EntTV.ts", Container: "ts", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "openlist", StoragePath: rootPath + "/ViuTV.940920.E02.HDTV.1080i.H264-EntTV.ts", Container: "ts", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library/电视剧", nil)

	seriesKey := recognition.SeriesWorkKey("ViuTV 940920")
	seasonKey := recognition.SeasonWorkKey("ViuTV 940920", 1)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "ViuTV 940920", SeasonNumber: 1, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "ViuTV 940920", SeasonNumber: 1, EpisodeNumber: 2})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertNoMovieWorkForTitle(t, output.Candidates, "ViuTV 940920")
}

func TestBuildScanRecognitionManifestOutputUsesSeparatedSeasonEpisodePatternAsSeries(t *testing.T) {
	rootPath := "/library/电视剧/太空部队 第二季[全7集][中文字幕].Space.Force.S02.1080p.Netflix.WEB-DL.H265.DDP5.1-SeeWEB"
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "openlist", StoragePath: rootPath + "/Space.Force.S02.E1.THE.INQUIRY.1080p.Netflix.WEB-DL.H265.DDP5.1-SeeWEB.mp4", Container: "mp4", ContentClass: SourceContentClassVideo, Status: "available"},
		{ID: 2, StorageProvider: "openlist", StoragePath: rootPath + "/Space.Force.S02.E2.BUDGET.CUTS.1080p.Netflix.WEB-DL.H265.DDP5.1-SeeWEB.mp4", Container: "mp4", ContentClass: SourceContentClassVideo, Status: "available"},
	}

	output := buildScanRecognitionManifestOutput(files, "/library/电视剧", nil)

	seriesKey := recognition.SeriesWorkKey("Space Force")
	seasonKey := recognition.SeasonWorkKey("Space Force", 2)
	episodeOneKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Space Force", SeasonNumber: 2, EpisodeNumber: 1})
	episodeTwoKey := recognition.EpisodeKey(recognition.EpisodeInput{SeriesTitle: "Space Force", SeasonNumber: 2, EpisodeNumber: 2})
	assertHasCandidate(t, output.Candidates, seriesKey, recognition.CandidateTypeWork, recognition.WorkKindSeries)
	assertHasCandidate(t, output.Candidates, seasonKey, recognition.CandidateTypeWork, recognition.WorkKindSeason)
	assertHasCandidate(t, output.Candidates, episodeOneKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertHasCandidate(t, output.Candidates, episodeTwoKey, recognition.CandidateTypeEpisode, recognition.WorkKindEpisode)
	assertNoMovieWorkForTitle(t, output.Candidates, "Space Force")
}

func TestBuildNewRecognitionSchemeOutputUsesConsistentMovieNFOForWeakMovieFilename(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Movies/Inception/file.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	sidecars := map[uint][]recognition.SidecarHint{
		1: {{Path: "/library/Movies/Inception/movie.nfo", Extension: ".nfo", ParseStatus: "parsed", MediaType: recognition.WorkKindMovie, Title: "Inception", Year: intPtrForReduction(2010)}},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", sidecars)

	movieKey := recognition.MovieWorkKey(recognition.MovieWorkInput{Title: "Inception", Year: intPtrForReduction(2010)})
	assertHasCandidate(t, output.Candidates, movieKey, recognition.CandidateTypeWork, recognition.WorkKindMovie)
	assertHasResourceParent(t, output.Candidates, movieKey, files[0].ID)
}

func TestBuildNewRecognitionSchemeOutputDoesNotMaterializeConflictingSidecar(t *testing.T) {
	files := []database.InventoryFile{
		{ID: 1, StorageProvider: "local", StoragePath: "/library/Shows/Six Feet Under/Season 1/Six.Feet.Under.S01E01.mkv", Container: "mkv", ContentClass: SourceContentClassVideo, Status: "available"},
	}
	sidecars := map[uint][]recognition.SidecarHint{
		1: {{Path: "/library/Shows/Six Feet Under/Season 1/movie.nfo", Extension: ".nfo", ParseStatus: "parsed", MediaType: recognition.WorkKindMovie, Title: "Six Feet Under", Year: intPtrForReduction(2001)}},
	}

	output := buildScanRecognitionManifestOutput(files, "/library", sidecars)

	if len(output.Candidates) != 0 {
		t.Fatalf("expected conflicting sidecar evidence not to materialize candidates, got %#v", output.Candidates)
	}
	assertEvidenceSource(t, output.Evidence, "scanrecognition")
}

func assertHasCandidate(t *testing.T, candidates []database.RecognitionCandidate, key string, candidateType string, role string) {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.CandidateKey == key && candidate.CandidateType == candidateType && candidate.CandidateRole == role {
			return
		}
	}
	t.Fatalf("expected candidate %s/%s/%s, got %#v", key, candidateType, role, candidates)
}

func assertHasResourceParent(t *testing.T, candidates []database.RecognitionCandidate, parentKey string, fileID uint) {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.CandidateType == recognition.CandidateTypePlayableResource && candidate.ParentCandidateKey == parentKey && candidate.PrimaryInventoryID != nil && *candidate.PrimaryInventoryID == fileID {
			return
		}
	}
	t.Fatalf("expected resource for file %d parented by %s, got %#v", fileID, parentKey, candidates)
}

func assertEvidenceSource(t *testing.T, evidence []database.RecognitionEvidence, source string) {
	t.Helper()
	for _, item := range evidence {
		if item.EvidenceSource == source {
			return
		}
	}
	t.Fatalf("expected evidence source %q, got %#v", source, evidence)
}

func assertNoMovieWorkForTitle(t *testing.T, candidates []database.RecognitionCandidate, title string) {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.CandidateType == recognition.CandidateTypeWork && candidate.CandidateRole == recognition.WorkKindMovie && strings.Contains(candidate.CandidateKey, strings.ToLower(title)) {
			t.Fatalf("expected no movie work for %q, got %#v", title, candidate)
		}
	}
}
