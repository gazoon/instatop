package competition

import (
	"context"
	"instarate/libs/instagram"
	"time"

	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils"
	"github.com/gazoon/go-utils/logging"
	"github.com/pkg/errors"
)

const (
	CelebritiesCompetition = "celebrities"
	ModelsCompetition      = "models"
	RegularCompetition     = "regular"
	GlobalCompetition      = "global"
)

const (
	httpTimeout = time.Second * 3

	celebrityFollowersThreshold = 500000
	modelFollowersThreshold     = 10000

	nextPairGetAttempts = 10
)

var (
	AlreadyVotedErr          = errors.New("already voted")
	BadPhotoLinkErr          = errors.New("photo link doesn't contain a valid media code")
	BadProfileLinkErr        = errors.New("profile link doesn't contain a valid username")
	NotPhotoMediaErr         = errors.New("media is not a photo")
	GetNextPairNoAttemptsErr = errors.New("out of attempts to get next pair")
)

type CompetitorNotFound struct {
	Username string
}

func (self *CompetitorNotFound) Error() string {
	return fmt.Sprintf("competitor %s doesn't exist", self.Username)
}

type InstCompetitor struct {
	Username string
	*InstProfile
	*Competitor
}

func (self InstCompetitor) String() string {
	return utils.ObjToString(&self)
}

type Competition struct {
	*logging.LoggerMixin
	competitors   *CompetitorsStorage
	profiles      *ProfilesStorage
	voters        *VotersStorage
	photosStorage *GoogleFilesStorage
}

func NewCompetition(competitors *CompetitorsStorage, profiles *ProfilesStorage,
	filesStorage *GoogleFilesStorage, voters *VotersStorage) *Competition {

	return &Competition{
		competitors: competitors, profiles: profiles, photosStorage: filesStorage,
		voters: voters, LoggerMixin: logging.NewLoggerMixin("competition", nil)}
}

func (self *Competition) GetPhotoUrl(competitor *InstCompetitor) string {
	return self.photosStorage.BuildUrl(competitor.PhotoPath)
}

func (self *Competition) GetPosition(ctx context.Context, competitor *InstCompetitor) (int, error) {
	num, err := self.competitors.GetNumberWithHigherRating(ctx, competitor.CompetitionCode, competitor.Rating)
	if err != nil {
		return 0, err
	}
	return num + 1, nil
}

func (self *Competition) Add(ctx context.Context, photoLink string) (*InstProfile, error) {
	logger := self.GetLogger(ctx)
	logger.WithField("photo_link", photoLink).Info("Add instagram competitor by photo link")
	mediaCode, err := utils.ExtractLastPathPart(photoLink)
	if err != nil {
		logger.WithFields(log.Fields{"photo_link": photoLink, "error": err}).
			Warn("Can't extract media code from the photo link")
		return nil, BadPhotoLinkErr
	}
	mediaInfo, err := instagram.GetMediaInfo(ctx, mediaCode)
	if err != nil {
		if err == instagram.MediaForbidden {
			logger.WithField("media_code", mediaCode).Warn("Media not found")
		}
		return nil, err
	}
	if !mediaInfo.IsPhoto {
		logger.WithField("media_code", mediaCode).Warn("Media is not a photo")
		return nil, NotPhotoMediaErr
	}
	followers, err := instagram.GetFollowersNumber(ctx, mediaInfo.Owner)
	if err != nil {
		return nil, err
	}
	profile := NewProfile(mediaInfo.Owner, mediaCode, followers)
	logger.WithField("profile", profile).Info("Add new instagram profile")
	err = self.profiles.Create(ctx, profile)
	if err == ProfileExistsErr {
		logger.WithField("username", mediaInfo.Owner).Info("Instagram profile already exists")
		return profile, ProfileExistsErr
	}
	if err != nil {
		return nil, err
	}
	_, err = self.photosStorage.Upload(ctx, profile.PhotoPath, mediaInfo.Url)
	if err != nil {
		return nil, err
	}
	competitions := ChoseCompetitions(followers)
	logger.WithField("competitions", competitions).Info("Add new profile to suitable competitions")
	for _, competitionCode := range competitions {
		compttr := NewCompetitor(mediaInfo.Owner, competitionCode)
		err = self.competitors.Create(ctx, compttr)
		if _, ok := err.(CompetitorAlreadyExists); ok {
			return nil, errors.Wrap(err, "competitor already allocated in the competition")
		}
		if err != nil {
			return nil, err
		}
	}
	return profile, nil
}

func (self *Competition) GetCompetitorsNumber(ctx context.Context, competitionCode string) (int, error) {
	return self.competitors.GetCompetitorsNumber(ctx, competitionCode)
}

func (self *Competition) GetNextPair(ctx context.Context, competitionCode, votersGroupId string) (*InstCompetitor, *InstCompetitor, error) {
	for i := 0; i < nextPairGetAttempts; i++ {
		competitor1, competitor2, err := self.competitors.GetRandomPair(ctx, competitionCode)
		if err != nil {
			return nil, nil, err
		}
		haveSeenPair, err := self.voters.HaveSeenPair(ctx, competitionCode, votersGroupId, competitor1.Username, competitor2.Username)
		if err != nil {
			return nil, nil, err
		}
		if haveSeenPair {
			continue
		}
		return self.convertPairToInstCompetitors(ctx, competitor1, competitor2)
	}
	return nil, nil, GetNextPairNoAttemptsErr
}

func (self *Competition) GetCompetitor(ctx context.Context, competitionCode, profileLink string) (*InstCompetitor, error) {
	username, err := utils.ExtractLastPathPart(profileLink)
	if err != nil {
		logger := self.GetLogger(ctx)
		logger.WithFields(log.Fields{"profile_link": profileLink, "error": err}).
			Warn("Can't extract username from the profile link")
		return nil, BadProfileLinkErr
	}
	compttr, err := self.competitors.Get(ctx, competitionCode, username)
	if err != nil {
		return nil, err
	}
	profile, err := self.profiles.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	return combineProfileAndCompetitor(profile, compttr), nil
}

func (self *Competition) GetRandomCompetitor(ctx context.Context, competitionCode string) (*InstCompetitor, error) {
	compttr, err := self.competitors.GetRandomOne(ctx, competitionCode)
	if err != nil {
		return nil, err
	}
	profile, err := self.profiles.Get(ctx, compttr.Username)
	if err != nil {
		return nil, err
	}
	return combineProfileAndCompetitor(profile, compttr), nil
}

func (self *Competition) Remove(ctx context.Context, usernames ...string) error {
	var err error
	for i := range usernames {
		usernames[i], err = utils.ExtractLastPathPart(usernames[i])
		if err != nil {
			return err
		}
	}
	err = self.competitors.Delete(ctx, usernames)
	if err != nil {
		return err
	}
	err = self.profiles.Delete(ctx, usernames)
	if err != nil {
		return err
	}
	return nil
}

func (self *Competition) GetTop(ctx context.Context, competitionCode string, number, offset int) ([]*InstCompetitor, error) {
	competitors, err := self.competitors.GetTop(ctx, competitionCode, number, offset)
	if err != nil {
		return nil, err
	}
	return self.convertToInstCompetitors(ctx, competitors...)
}

func (self *Competition) Vote(ctx context.Context, competitionCode, votersGroupId, voterId, winnerUsername, loserUsername string) (*InstCompetitor, *InstCompetitor, error) {
	logger := self.GetLogger(ctx)
	logger.WithFields(log.Fields{
		"competition":     competitionCode,
		"voters_group_id": votersGroupId,
		"voter_id":        voterId,
		"winner":          winnerUsername,
		"loser":           loserUsername,
	}).Info("Save user vote")
	ok, err := self.voters.TryVote(ctx, competitionCode, votersGroupId, voterId, winnerUsername, loserUsername)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		logger.Info("User already voted")
		return nil, nil, AlreadyVotedErr
	}

	winner, err := self.competitors.Get(ctx, competitionCode, winnerUsername)
	if err != nil {
		return nil, nil, err
	}
	loser, err := self.competitors.Get(ctx, competitionCode, loserUsername)
	if err != nil {
		return nil, nil, err
	}

	winner.Rating, loser.Rating = recalculateEloRating(winner.Rating, loser.Rating)
	winner.Wins += 1
	winner.Matches += 1
	loser.Loses += 1
	loser.Matches += 1

	err = self.competitors.Update(ctx, winner)
	if err != nil {
		return nil, nil, err
	}
	err = self.competitors.Update(ctx, loser)
	if err != nil {
		return nil, nil, err
	}

	return self.convertPairToInstCompetitors(ctx, winner, loser)
}

func (self *Competition) convertPairToInstCompetitors(ctx context.Context, competitor1, competitor2 *Competitor) (*InstCompetitor, *InstCompetitor, error) {
	competitorsPair, err := self.convertToInstCompetitors(ctx, competitor1, competitor2)
	if err != nil {
		return nil, nil, err
	}
	if len(competitorsPair) != 2 {
		return nil, nil, errors.Errorf("expected two competitors, got: %v", competitorsPair)
	}
	return competitorsPair[0], competitorsPair[1], nil
}

func (self *Competition) convertToInstCompetitors(ctx context.Context, competitors ...*Competitor) ([]*InstCompetitor, error) {
	usernames := make([]string, len(competitors))
	for i := range usernames {
		usernames[i] = competitors[i].Username
	}
	profiles, err := self.profiles.GetMultiple(ctx, usernames)
	if err != nil {
		return nil, err
	}
	profilesMapping := make(map[string]*InstProfile, len(profiles))
	for _, profile := range profiles {
		profilesMapping[profile.Username] = profile
	}
	if len(competitors) != len(profilesMapping) {
		return nil, errors.New("number of profiles is not equal to number of competitors")
	}
	result := make([]*InstCompetitor, len(competitors))
	for i, competitor := range competitors {
		profile, ok := profilesMapping[competitor.Username]
		if !ok {
			return nil, errors.Errorf("cant find profile for competitor %s", competitor.Username)
		}
		result[i] = combineProfileAndCompetitor(profile, competitor)
	}
	return result, nil
}

func combineProfileAndCompetitor(profile *InstProfile, competitor *Competitor) *InstCompetitor {
	return &InstCompetitor{InstProfile: profile, Competitor: competitor, Username: profile.Username}
}

func ChoseCompetitions(followersNumber int) []string {
	var competitionByFollowers string
	if followersNumber < modelFollowersThreshold {
		competitionByFollowers = RegularCompetition
	} else if followersNumber < celebrityFollowersThreshold {
		competitionByFollowers = ModelsCompetition
	} else {
		competitionByFollowers = CelebritiesCompetition
	}
	return []string{GlobalCompetition, competitionByFollowers}
}
