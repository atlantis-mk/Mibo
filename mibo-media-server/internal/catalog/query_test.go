package catalog

import (
	"testing"
	"time"

	"github.com/atlan/mibo-media-server/internal/database"
)

func TestCatalogQueryAPIsReturnDetailAndGovernanceWorkspace(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA", SortKey: "Show A", AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceNeedsReview})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/shows/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeNumber := 2
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if _, err := svc.RecordMetadataSource(ctx, MetadataSourceInput{ItemID: series.ID, SourceType: SourceTypeProvider, SourceName: "tmdb", ExternalID: "tv:777", PayloadJSON: `{"title":"Show A"}`}); err != nil {
		t.Fatalf("record source: %v", err)
	}
	if _, err := svc.SetExternalID(ctx, ExternalIDInput{ItemID: series.ID, Provider: "tmdb", ProviderType: "tv", ExternalID: "tv:777", IsPrimary: true}); err != nil {
		t.Fatalf("set external id: %v", err)
	}
	if _, _, err := svc.ApplyField(ctx, ApplyFieldInput{ItemID: series.ID, FieldKey: "title", Value: "Show A", Lock: true, LockReason: "manual"}); err != nil {
		t.Fatalf("apply field: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.ItemImage{ItemID: series.ID, ImageType: "poster", URL: "https://example.com/poster.jpg", IsSelected: true}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	related, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Related Show", Path: "/shows/Related", SortKey: "Related Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create related item: %v", err)
	}
	genre := database.Tag{Kind: "genre", Name: "Drama"}
	topic := database.Tag{Kind: "topic", Name: "Space"}
	if err := svc.db.WithContext(ctx).Create([]*database.Tag{&genre, &topic}).Error; err != nil {
		t.Fatalf("create tags: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemTag{
		{ItemID: series.ID, TagID: genre.ID},
		{ItemID: series.ID, TagID: topic.ID},
		{ItemID: related.ID, TagID: genre.ID},
	}).Error; err != nil {
		t.Fatalf("link tags: %v", err)
	}
	actor := database.Person{Name: "Actor A", SortName: "actor a", AvatarURL: "https://example.com/actor-a.jpg"}
	director := database.Person{Name: "Director A", SortName: "director a", AvatarURL: "https://example.com/director-a.jpg"}
	if err := svc.db.WithContext(ctx).Create([]*database.Person{&actor, &director}).Error; err != nil {
		t.Fatalf("create people: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemPerson{{ItemID: series.ID, PersonID: actor.ID, Role: "cast", Character: "Lead", SortOrder: 0}, {ItemID: series.ID, PersonID: director.ID, Role: "director", Character: "Director", SortOrder: 0}}).Error; err != nil {
		t.Fatalf("link people: %v", err)
	}

	items, err := svc.ListLibraryItems(ctx, 1, "Show A", "show", 10)
	if err != nil {
		t.Fatalf("list library items: %v", err)
	}
	if len(items) != 1 || items[0].ID != series.ID || items[0].Type != ItemTypeSeries {
		t.Fatalf("unexpected library items: %#v", items)
	}

	detail, err := svc.GetItemDetail(ctx, series.ID)
	if err != nil {
		t.Fatalf("get item detail: %v", err)
	}
	if detail.ID != series.ID || len(detail.Seasons) != 1 || len(detail.Seasons[0].Episodes) != 1 || detail.Seasons[0].Episodes[0].ID != episode.ID {
		t.Fatalf("unexpected item detail: %#v", detail)
	}
	if len(detail.Cast) != 1 || detail.Cast[0].ID != actor.ID || detail.Cast[0].Name != "Actor A" || detail.Cast[0].Role != "Lead" || detail.Cast[0].AvatarURL != "https://example.com/actor-a.jpg" {
		t.Fatalf("unexpected cast detail: %#v", detail.Cast)
	}
	if len(detail.Directors) != 1 || detail.Directors[0].ID != director.ID || detail.Directors[0].Name != "Director A" || detail.Directors[0].Role != "Director" || detail.Directors[0].AvatarURL != "https://example.com/director-a.jpg" {
		t.Fatalf("unexpected directors detail: %#v", detail.Directors)
	}
	if len(detail.Tags) != 2 || detail.Tags[0].Kind != "genre" || detail.Tags[0].Name != "Drama" || detail.Tags[1].Name != "Space" {
		t.Fatalf("unexpected tags detail: %#v", detail.Tags)
	}
	if len(detail.Genres) != 1 || detail.Genres[0] != "Drama" {
		t.Fatalf("unexpected genres detail: %#v", detail.Genres)
	}
	if len(detail.RelatedItems) == 0 || detail.RelatedItems[0].ID != related.ID {
		t.Fatalf("unexpected related detail: %#v", detail.RelatedItems)
	}

	seasons, err := svc.ListSeriesSeasons(ctx, series.ID)
	if err != nil {
		t.Fatalf("list series seasons: %v", err)
	}
	if len(seasons) != 1 || len(seasons[0].Episodes) != 1 || seasons[0].Episodes[0].ID != episode.ID {
		t.Fatalf("unexpected seasons payload: %#v", seasons)
	}

	workspace, err := svc.GetGovernanceWorkspace(ctx, series.ID)
	if err != nil {
		t.Fatalf("get governance workspace: %v", err)
	}
	if workspace.ItemID != series.ID || len(workspace.SourceEvidence) != 1 || len(workspace.FieldStates) != 1 || len(workspace.SelectedImages) != 1 || len(workspace.RecommendedChildren) != 1 {
		t.Fatalf("unexpected governance workspace: %#v", workspace)
	}
}

func TestGetPersonDetailReturnsProfileAndOrderedRelatedWorks(t *testing.T) {
	svc, ctx := newTestService(t)
	birthday := time.Date(1988, 5, 4, 0, 0, 0, 0, time.UTC)
	tmdbPersonID := 321
	person := database.Person{
		Name:               "Actor A",
		SortName:           "Actor A",
		AvatarURL:          "https://example.com/actor-a.jpg",
		TMDBPersonID:       &tmdbPersonID,
		IMDBID:             "nm0000321",
		Biography:          "Lead performer.",
		Birthday:           &birthday,
		PlaceOfBirth:       "Seoul",
		KnownForDepartment: "Acting",
	}
	if err := svc.db.WithContext(ctx).Create(&person).Error; err != nil {
		t.Fatalf("create person: %v", err)
	}

	missingYear := 2025
	availableOlderYear := 2021
	availableNewerYear := 2024
	missing, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Missing Movie", Path: "/movies/missing.mkv", SortKey: "Missing Movie", Year: &missingYear, AvailabilityStatus: AvailabilityMissing, GovernanceStatus: GovernanceMatched})
	if err != nil {
		t.Fatalf("create missing item: %v", err)
	}
	availableOlder, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Older Movie", Path: "/movies/older.mkv", SortKey: "Older Movie", Year: &availableOlderYear, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched})
	if err != nil {
		t.Fatalf("create older item: %v", err)
	}
	availableNewer, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Newer Movie", Path: "/movies/newer.mkv", SortKey: "Newer Movie", Year: &availableNewerYear, AvailabilityStatus: AvailabilityAvailable, GovernanceStatus: GovernanceMatched})
	if err != nil {
		t.Fatalf("create newer item: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemImage{{ItemID: availableNewer.ID, ImageType: "backdrop", URL: "https://example.com/newer-backdrop.jpg", IsSelected: true}}).Error; err != nil {
		t.Fatalf("create related item image: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemPerson{{ItemID: missing.ID, PersonID: person.ID, Role: "cast", Character: "Guest", SortOrder: 0}, {ItemID: availableOlder.ID, PersonID: person.ID, Role: "cast", Character: "Support", SortOrder: 1}, {ItemID: availableNewer.ID, PersonID: person.ID, Role: "cast", Character: "Lead", SortOrder: 1}}).Error; err != nil {
		t.Fatalf("link related items: %v", err)
	}

	detail, err := svc.GetPersonDetail(ctx, person.ID)
	if err != nil {
		t.Fatalf("get person detail: %v", err)
	}
	if detail.ID != person.ID || detail.Name != person.Name || detail.Biography != person.Biography || detail.Birthday == nil || !detail.Birthday.Equal(birthday) {
		t.Fatalf("unexpected person detail profile: %#v", detail)
	}
	if len(detail.ExternalIdentities) != 2 || detail.ExternalIdentities[0].Provider != "tmdb" || detail.ExternalIdentities[0].ProviderType != "person" || detail.ExternalIdentities[0].ExternalID != "321" || detail.ExternalIdentities[1].Provider != "imdb" {
		t.Fatalf("unexpected external identities: %#v", detail.ExternalIdentities)
	}
	if len(detail.RelatedItems) != 3 || detail.RelatedItems[0].ID != availableNewer.ID || detail.RelatedItems[1].ID != availableOlder.ID || detail.RelatedItems[2].ID != missing.ID {
		t.Fatalf("unexpected related item order: %#v", detail.RelatedItems)
	}
}

func TestGetEpisodeItemDetailIncludesContextShelfProgressAndStreams(t *testing.T) {
	svc, ctx := newTestService(t)
	series, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Show A", Path: "/shows/ShowA", SortKey: "Show A", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	seasonNumber := 1
	season, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeason, ParentID: &series.ID, Title: "Season 1", Path: "/shows/ShowA/Season 1", SortKey: "Show A S01", IndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create season: %v", err)
	}
	episodeOneNumber := 1
	episodeOne, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 1", Path: "/shows/ShowA/Season 1/ShowA.S01E01.mkv", SortKey: "Show A S01E01", IndexNumber: &episodeOneNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode one: %v", err)
	}
	episodeTwoNumber := 2
	episodeTwo, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, ParentID: &season.ID, Title: "Episode 2", Path: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SortKey: "Show A S01E02", IndexNumber: &episodeTwoNumber, ParentIndexNumber: &seasonNumber, RuntimeSeconds: intPtr(1800), AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create episode two: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create([]database.ItemImage{
		{ItemID: series.ID, ImageType: "backdrop", URL: "https://example.com/series.jpg", IsSelected: true},
		{ItemID: season.ID, ImageType: "poster", URL: "https://example.com/season.jpg", IsSelected: true},
		{ItemID: episodeOne.ID, ImageType: "still", URL: "https://example.com/e1.jpg", IsSelected: true},
		{ItemID: episodeTwo.ID, ImageType: "still", URL: "https://example.com/e2.jpg", IsSelected: true},
	}).Error; err != nil {
		t.Fatalf("create images: %v", err)
	}

	asset := database.MediaAsset{LibraryID: 1, AssetType: "main", DisplayName: "1080p", Status: AvailabilityAvailable, ProbeStatus: "ready"}
	if err := svc.db.WithContext(ctx).Create(&asset).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetItem{AssetID: asset.ID, ItemID: episodeTwo.ID, Role: "primary", SegmentIndex: 0}).Error; err != nil {
		t.Fatalf("link asset: %v", err)
	}
	file := database.InventoryFile{LibraryID: 1, StorageProvider: "local", StoragePath: "/shows/ShowA/Season 1/ShowA.S01E02.mkv", SizeBytes: 123456, Container: "mkv", Status: AvailabilityAvailable}
	if err := svc.db.WithContext(ctx).Create(&file).Error; err != nil {
		t.Fatalf("create inventory file: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&database.AssetFile{AssetID: asset.ID, FileID: file.ID, Role: "source", PartIndex: 0}).Error; err != nil {
		t.Fatalf("link asset file: %v", err)
	}
	width := 1920
	height := 1080
	level := 41
	bitDepth := 10
	referenceFrames := 4
	channels := 6
	bitrate := int64(640000)
	videoBitrate := int64(4200000)
	audioBitDepth := 24
	audioSampleRate := 48000
	if err := svc.db.WithContext(ctx).Create([]database.MediaStream{
		{FileID: file.ID, StreamIndex: 0, StreamType: "video", Codec: "h264", Profile: "High", Level: &level, Width: &width, Height: &height, AvgFrameRate: "24000/1001", RFrameRate: "24000/1001", FieldOrder: "progressive", ColorSpace: "bt709", BitDepth: &bitDepth, PixelFormat: "yuv420p10le", ReferenceFrames: &referenceFrames, BitRate: &videoBitrate},
		{FileID: file.ID, StreamIndex: 1, StreamType: "audio", Codec: "flac", Language: "jpn", Title: "Japanese", Channels: &channels, ChannelLayout: "5.1(side)", SampleRate: &audioSampleRate, BitDepth: &audioBitDepth, BitRate: &bitrate, DispositionJSON: `{"default":true}`},
		{FileID: file.ID, StreamIndex: 2, StreamType: "subtitle", Codec: "ass", Language: "zho", Title: "Chinese Traditional", DispositionJSON: `{"default":true,"forced":false,"external":true,"hearing_impaired":false}`},
	}).Error; err != nil {
		t.Fatalf("create streams: %v", err)
	}
	playedPercentage := 55.5
	if err := svc.db.WithContext(ctx).Create(&database.UserItemData{UserID: 7, ItemID: episodeOne.ID, PositionSeconds: 600, PlayedPercentage: &playedPercentage}).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}

	detail, err := svc.GetItemDetailForUser(ctx, episodeTwo.ID, uintPtr(7))
	if err != nil {
		t.Fatalf("get episode detail: %v", err)
	}
	if detail.EpisodeContext == nil || detail.EpisodeContext.Series == nil || detail.EpisodeContext.Series.ID != series.ID || detail.EpisodeContext.Season == nil || detail.EpisodeContext.Season.ID != season.ID {
		t.Fatalf("unexpected episode context: %#v", detail.EpisodeContext)
	}
	if detail.EpisodeContext.IncompleteHierarchy {
		t.Fatalf("expected complete hierarchy context: %#v", detail.EpisodeContext)
	}
	if detail.EpisodeContext.EpisodeNumber == nil || *detail.EpisodeContext.EpisodeNumber != episodeTwoNumber || len(detail.EpisodeContext.Series.SelectedImages) != 1 || len(detail.EpisodeContext.Season.SelectedImages) != 1 {
		t.Fatalf("unexpected episode numbering or parent images: %#v", detail.EpisodeContext)
	}
	if len(detail.SameSeasonEpisodes) != 2 || detail.SameSeasonEpisodes[0].ID != episodeOne.ID || detail.SameSeasonEpisodes[0].Progress == nil || detail.SameSeasonEpisodes[0].Progress.PositionSeconds != 600 {
		t.Fatalf("unexpected same-season shelf progress: %#v", detail.SameSeasonEpisodes)
	}
	if !detail.SameSeasonEpisodes[1].Current || detail.SameSeasonEpisodes[1].Label != "S1:E2" || detail.SameSeasonEpisodes[1].Progress != nil {
		t.Fatalf("unexpected current episode shelf state: %#v", detail.SameSeasonEpisodes[1])
	}
	if len(detail.Assets) != 1 || len(detail.Assets[0].Files) != 1 || detail.Assets[0].Files[0].FileID != file.ID || detail.Assets[0].Files[0].Container != "mkv" {
		t.Fatalf("unexpected asset file summaries: %#v", detail.Assets)
	}
	if len(detail.Assets[0].Streams) != 3 || detail.Assets[0].Streams[0].Width == nil || *detail.Assets[0].Streams[0].Width != width || !detail.Assets[0].Streams[1].Default {
		t.Fatalf("unexpected asset stream summaries: %#v", detail.Assets[0].Streams)
	}
	videoStream := detail.Assets[0].Streams[0]
	if videoStream.Profile != "High" || videoStream.Level == nil || *videoStream.Level != level || videoStream.AvgFrameRate != "24000/1001" || videoStream.BitDepth == nil || *videoStream.BitDepth != bitDepth || videoStream.ReferenceFrames == nil || *videoStream.ReferenceFrames != referenceFrames || videoStream.BitRate == nil || *videoStream.BitRate != videoBitrate {
		t.Fatalf("unexpected detailed video stream summary: %#v", videoStream)
	}
	if videoStream.FieldOrder != "progressive" || videoStream.ColorSpace != "bt709" || videoStream.PixelFormat != "yuv420p10le" {
		t.Fatalf("unexpected detailed video stream display fields: %#v", videoStream)
	}
	audioStream := detail.Assets[0].Streams[1]
	if audioStream.ChannelLayout != "5.1(side)" || audioStream.SampleRate == nil || *audioStream.SampleRate != audioSampleRate || audioStream.BitDepth == nil || *audioStream.BitDepth != audioBitDepth || audioStream.Codec != "flac" {
		t.Fatalf("unexpected detailed audio stream summary: %#v", audioStream)
	}
	subtitleStream := detail.Assets[0].Streams[2]
	if subtitleStream.Codec != "ass" || subtitleStream.Title != "Chinese Traditional" || !subtitleStream.Default || subtitleStream.Forced || !subtitleStream.External || subtitleStream.HearingImpaired {
		t.Fatalf("unexpected detailed subtitle stream summary: %#v", subtitleStream)
	}
}

func TestGetEpisodeItemDetailAllowsIncompleteHierarchy(t *testing.T) {
	svc, ctx := newTestService(t)
	seasonNumber := 1
	episodeNumber := 2
	episode, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeEpisode, Title: "Loose Episode", Path: "/shows/Loose.S01E02.mkv", SortKey: "Loose S01E02", IndexNumber: &episodeNumber, ParentIndexNumber: &seasonNumber, AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create loose episode: %v", err)
	}

	detail, err := svc.GetItemDetail(ctx, episode.ID)
	if err != nil {
		t.Fatalf("get episode detail: %v", err)
	}
	if detail.EpisodeContext == nil || !detail.EpisodeContext.IncompleteHierarchy || detail.EpisodeContext.Series != nil || detail.EpisodeContext.Season != nil {
		t.Fatalf("expected incomplete hierarchy context, got %#v", detail.EpisodeContext)
	}
	if len(detail.SameSeasonEpisodes) != 0 {
		t.Fatalf("expected no same-season shelf for loose episode, got %#v", detail.SameSeasonEpisodes)
	}
}

func TestUserItemFavoritesAndContinueWatching(t *testing.T) {
	svc, ctx := newTestService(t)
	movie, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Favorite Movie", Path: "/movies/favorite.mkv", SortKey: "Favorite Movie", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	show, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeSeries, Title: "Watching Show", Path: "/shows/watching", SortKey: "Watching Show", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create show: %v", err)
	}

	const userID uint = 7
	favorite, err := svc.SetFavorite(ctx, userID, movie.ID, true)
	if err != nil {
		t.Fatalf("set favorite: %v", err)
	}
	if !favorite.Favorite || favorite.Item.ID != movie.ID {
		t.Fatalf("unexpected favorite response: %#v", favorite)
	}

	favorites, err := svc.ListFavorites(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list favorites: %v", err)
	}
	if len(favorites) != 1 || favorites[0].Item.ID != movie.ID {
		t.Fatalf("unexpected favorites: %#v", favorites)
	}

	lastPlayed := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&database.UserItemData{UserID: userID, ItemID: show.ID, PositionSeconds: 120, LastPlayedAt: &lastPlayed}).Error; err != nil {
		t.Fatalf("create progress: %v", err)
	}
	continueWatching, err := svc.ListContinueWatching(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list continue watching: %v", err)
	}
	if len(continueWatching) != 1 || continueWatching[0].Item.ID != show.ID || continueWatching[0].PositionSeconds != 120 {
		t.Fatalf("unexpected continue watching: %#v", continueWatching)
	}

	if _, err := svc.SetFavorite(ctx, userID, movie.ID, false); err != nil {
		t.Fatalf("remove favorite: %v", err)
	}
	favorites, err = svc.ListFavorites(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list favorites after remove: %v", err)
	}
	if len(favorites) != 0 {
		t.Fatalf("expected no favorites, got %#v", favorites)
	}
}

func TestGetItemDetailFallsBackToLegacyPeopleWhenCatalogPeopleAreMissing(t *testing.T) {
	svc, ctx := newTestService(t)
	item, err := svc.CreateItem(ctx, CreateItemInput{LibraryID: 1, Type: ItemTypeMovie, Title: "Movie A", Path: "/movies/MovieA.2024.mkv", SortKey: "Movie A", AvailabilityStatus: AvailabilityAvailable})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	legacy := database.MediaItem{
		LibraryID:        item.LibraryID,
		Type:             ItemTypeMovie,
		Title:            "Movie A",
		SourcePath:       item.Path,
		CastJSON:         `[{"name":"Actor A","role":"Lead"}]`,
		DirectorsJSON:    `[{"name":"Director A","role":"Director"}]`,
		MatchStatus:      "matched",
		MetadataProvider: "tmdb",
		Status:           "ready",
	}
	if err := svc.db.WithContext(ctx).Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy item: %v", err)
	}

	detail, err := svc.GetItemDetail(ctx, item.ID)
	if err != nil {
		t.Fatalf("get item detail: %v", err)
	}
	if len(detail.Cast) != 1 || detail.Cast[0].Name != "Actor A" || detail.Cast[0].Role != "Lead" {
		t.Fatalf("unexpected fallback cast detail: %#v", detail.Cast)
	}
	if len(detail.Directors) != 1 || detail.Directors[0].Name != "Director A" || detail.Directors[0].Role != "Director" {
		t.Fatalf("unexpected fallback directors detail: %#v", detail.Directors)
	}
}
