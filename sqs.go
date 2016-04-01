package queue

import (
	"DaiZong/util"
	"DaiZong/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/Sirupsen/logrus"
)

type Queue struct {
	SQS *sqs.SQS
	URL *string
}

var log = logrus.New()

func NewQueue(queuename string, region string) (*Queue, error) {
	s := sqs.New(session.New(&aws.Config{
		Region: aws.String(region),
		Credentials:credentials.NewStaticCredentials(config.Conf.GetYamlValue("access_key"), config.Conf.GetYamlValue("secret_key"), ""),
	}))
	if queuename == "" {
		queuename = util.IpAddress()
	}
	u, err := GetQueueUrl(s, queuename)
	if err != nil {
		return nil, err
	}
	return &Queue{
		SQS: s,
		URL: u,
	}, nil
}

func GetQueueUrl(s *sqs.SQS, name string) (*string, error) {
	req := &sqs.GetQueueUrlInput{
		QueueName: &name,
	}
	resp, err := s.GetQueueUrl(req)
	if err != nil {
		resp, _ := CreatQueue(&name, s)
		return resp.QueueUrl, err
	}
	return resp.QueueUrl, nil
}

func CreatQueue(queuename *string, s *sqs.SQS) (*sqs.CreateQueueOutput, error) {
	req := &sqs.CreateQueueInput{
		QueueName: queuename,
	}
	resp, err := s.CreateQueue(req)
	return resp, err
}

func (q *Queue) ChangeMessageVisibility(recipthandler *string, visibilityTimeOut int64) error {
	req := &sqs.ChangeMessageVisibilityInput{
		ReceiptHandle:     recipthandler,
		QueueUrl:          q.URL,
		VisibilityTimeout: aws.Int64(visibilityTimeOut),
	}
	_, err := q.SQS.ChangeMessageVisibility(req)
	return err
}

func (q *Queue) SendMessage(message string) error {
	req := &sqs.SendMessageInput{
		MessageBody: aws.String(message),
		QueueUrl:    q.URL,
	}
	_, err := q.SQS.SendMessage(req)

	log.Info(err)
	return err
}

func (q *Queue) ReceiveMessage(visibilityTimeout int64, maxNum int64) (*sqs.ReceiveMessageOutput, error) {
	req := &sqs.ReceiveMessageInput{
		QueueUrl:            q.URL,
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
		MaxNumberOfMessages: aws.Int64(maxNum),
	}

	reponse, err := q.SQS.ReceiveMessage(req)

	if err != nil {
		return nil, err
	}
	return reponse, nil
}

func (q *Queue) DeleteMessage(recipthandler *string) (*sqs.DeleteMessageOutput, error) {

	req := &sqs.DeleteMessageInput{
		ReceiptHandle: recipthandler,
		QueueUrl:      q.URL,
	}
	out, err := q.SQS.DeleteMessage(req)
	return out, err
}

func (q *Queue) DeleteQueue() error {

	req := &sqs.DeleteQueueInput{
		QueueUrl: q.URL,
	}
	_, err := q.SQS.DeleteQueue(req)
	return err

}
