package relay

import (
	"net"
	"regexp"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pinpointemail"
	"github.com/aws/aws-sdk-go/service/pinpointemail/pinpointemailiface"
	"github.com/blueimp/aws-smtp-relay/internal/relay"
)

// Client implements the Relay interface.
type Client struct {
	pinpointAPI     pinpointemailiface.PinpointEmailAPI
	setName         *string
	allowFromRegExp *regexp.Regexp
	allowToRegExp   *regexp.Regexp
	denyToRegExp    *regexp.Regexp
	prependSubject  *string
	carbonCopy      *string
}

// Send uses the given Pinpoint API to send email data
func (c Client) Send(
	origin net.Addr,
	from string,
	to []string,
	data []byte,
) error {
	allowedRecipients, deniedRecipients, err := relay.FilterAddresses(
		from,
		to,
		c.allowFromRegExp,
		c.allowToRegExp,
		c.denyToRegExp,
	)
	if err != nil {
		relay.Log(origin, &from, deniedRecipients, err)
	}
	if c.prependSubject != nil {
		data = relay.PrependSubject(data, *c.prependSubject)
	}
	var ccAddresses = []*string{}
	if c.carbonCopy != nil && *c.carbonCopy != "" {
		ccAddresses = append(ccAddresses, c.carbonCopy)
	}
	if len(allowedRecipients) > 0 {
		_, err := c.pinpointAPI.SendEmail(&pinpointemail.SendEmailInput{
			ConfigurationSetName: c.setName,
			FromEmailAddress:     &from,
			Destination: &pinpointemail.Destination{
				ToAddresses: allowedRecipients,
				CcAddresses: ccAddresses,
			},
			Content: &pinpointemail.EmailContent{
				Raw: &pinpointemail.RawMessage{
					Data: data,
				},
			},
		})
		relay.Log(origin, &from, allowedRecipients, err)
		if err != nil {
			return err
		}
	}
	return err
}

// New creates a new client with a session.
func New(
	configurationSetName *string,
	allowFromRegExp *regexp.Regexp,
	allowToRegExp *regexp.Regexp,
	denyToRegExp *regexp.Regexp,
	prependSubject *string,
	carbonCopy *string,
) Client {
	return Client{
		pinpointAPI:     pinpointemail.New(session.Must(session.NewSession())),
		setName:         configurationSetName,
		allowFromRegExp: allowFromRegExp,
		allowToRegExp:   allowToRegExp,
		denyToRegExp:    denyToRegExp,
		prependSubject:  prependSubject,
		carbonCopy:      carbonCopy,
	}
}
