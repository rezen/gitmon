package gitmon

import (
	"fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ses"
    // "github.com/aws/aws-sdk-go/aws/awserr"


)

// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/ses-example-send-email.html

type Email struct {
    To string
    Subject string
    Template string
    Content string
    AddTracker bool
}

type Emailer interface {
	Send(to string, subject string, content string) error
}

type FakeEmailer struct {

}
func (f FakeEmailer) Send(to string, subject string, content string) error {
    fmt.Println("[fakemail]", to, subject, content)
    return nil
}

type EmailViaSes struct {
	Sender string
	Session *session.Session
	Service *ses.SES
	// Template ....
}

func CreateEmailer() *EmailViaSes {
	sesh, err := session.NewSession(&aws.Config{
        Region:aws.String("us-west-2")},
	)
	
	if err != nil {
		panic(err)
	}

	return &EmailViaSes{
		Sender: "user@email.com",
		Session: sesh,
		Service: ses.New(sesh),
	}
}


func (e *EmailViaSes) Send(to string, subject string, content string) error {
	input := &ses.SendEmailInput{
        Destination: &ses.Destination{
            CcAddresses: []*string{
            },
            ToAddresses: []*string{
                aws.String(to),
            },
        },
        Message: &ses.Message{
            Body: &ses.Body{
                Html: &ses.Content{
                    Charset: aws.String("UTF-8"),
                    Data:    aws.String(content),
                },
            },
            Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
                Data:    aws.String(subject),
            },
        },
        Source: aws.String(e.Sender),
	}
	
	result, err := e.Service.SendEmail(input)
	fmt.Println(result)
	return err
}