package s3

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/turnerlabs/udeploy/component/app"
	"github.com/turnerlabs/udeploy/component/version"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const ext = ".zip"

// ListDefinitions ...
func ListDefinitions(instance app.Instance) (map[string]app.Definition, error) {

	results, err := List(instance.S3RegistryBucket, instance.S3RegistryPrefix)
	if err != nil {
		return nil, err
	}

	results = limitRevisions(results, instance.Task.Revisions)

	svc := s3.New(session.New())

	versions := map[string]app.Definition{}

	for _, o := range results {

		result, err := svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(instance.S3RegistryBucket),
			Key:    aws.String(*o.Key),
		})
		if err != nil {
			log.Println(*o.Key)
			log.Println(err)
			continue
		}

		ver, revision, err := extractMetadata(result.Metadata)
		if err != nil {
			log.Printf("%s %s\n", *o.Key, err)
			continue
		}

		n, err := strconv.ParseInt(revision, 10, 64)
		if err != nil {
			log.Printf("%s %s\n", *o.Key, err)
			continue
		}

		v, b := version.Extract(ver, instance.Task.ImageTagEx)

		def := app.Definition{
			Version:  v,
			Build:    b,
			Revision: n,

			Environment: map[string]string{},
			Secrets:     map[string]string{},

			Description: v,

			Registry: true,
		}

		versions[def.FormatVersion()] = def
	}

	return versions, nil
}

// List ...
func List(bucket, registry string) ([]*s3.Object, error) {
	svc := s3.New(session.New())

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
	}

	if len(registry) > 0 {
		input.SetPrefix(registry)
	}

	result, err := svc.ListObjects(input)
	if err != nil {
		return []*s3.Object{}, err
	}

	zips := []*s3.Object{}

	for _, file := range result.Contents {
		if strings.Contains(*file.Key, ext) {
			zips = append(zips, file)
		}
	}

	return zips, nil
}

func limitRevisions(objs []*s3.Object, limit int) []*s3.Object {

	if limit == 0 || len(objs) <= limit {
		return objs
	}

	sort.Slice(objs, func(i, j int) bool {
		return objs[i].LastModified.After(*objs[j].LastModified)
	})

	return objs[0:limit]
}
