package art

const (
	musicBrainzReleaseEndpint    = "%s/ws/2/release/"
	musicBrainzReleaseQueryValue = "release:%s AND artist:%s"
)

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
