package art

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"
)

const (
	musicBrainzReleaseEndpint    = "%s/ws/2/release/"
	musicBrainzReleaseQueryValue = "release:%s AND artist:%s"
)

// getMusicBrainzReleaseID uses the MusicBrainz API to retrieve a list of matching
// MusicBrainzIDs (or mbid) for particular "release". Or album in HTTPMS parlance.
func (c *Client) getMusicBrainzReleaseID(
	ctx context.Context,
	artist,
	album string,
) ([]string, error) {
	c.Lock()
	defer c.Unlock()

	<-c.delayer.C
	defer c.delayer.Reset(c.delay)

	mbURL := fmt.Sprintf(musicBrainzReleaseEndpint, c.musicBrainzAPIHost)
	req, err := http.NewRequest(http.MethodGet, mbURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating music brainz XML API req: %w", err)
	}

	query := req.URL.Query()
	query.Add("query", fmt.Sprintf(musicBrainzReleaseQueryValue, album, artist))
	req.URL.RawQuery = query.Encode()
	req.Header.Set("User-Agent", c.useragent)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("music brainz XML API returned HTTP %d", resp.StatusCode)
	}

	root := mbReleaseMetadata{}
	dec := xml.NewDecoder(resp.Body)

	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decoding music brainz XML API response: %w", err)
	}

	if len(root.RelaseList.Relases) < 1 {
		return nil, ErrImageNotFound
	}

	var releaseIDs []string
	for _, release := range root.RelaseList.Relases {
		if release.Score >= c.MinScore {
			releaseIDs = append(releaseIDs, release.ID)
		}
	}

	if len(releaseIDs) < 1 {
		return nil, ErrImageNotFound
	}

	return releaseIDs, nil
}

// The following are structures only used to decode the XML response from MusicBrainz
// API. And only the stuff we are interested and nothing more.
type mbReleaseMetadata struct {
	RelaseList mbReleaseList `xml:"release-list"`
}

type mbReleaseList struct {
	Relases []mbRelease `xml:"release"`
}

type mbRelease struct {
	ID    string `xml:"id,attr"`
	Score int    `xml:"score,attr"`
	Title string `xml:"title"`
}
