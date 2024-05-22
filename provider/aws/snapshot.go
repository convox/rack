package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
)

func (p *Provider) SetDBDeletionProtectionAndCreateSnapShot(app, resource, snapshot string) (string, error) {
	dbIdentifier, err := p.getResourceDBIdentifier(app, resource)
	if err != nil {
		return "", fmt.Errorf("db error: %s", err)
	}

	input := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbIdentifier),
		DeletionProtection:   aws.Bool(true),
	}

	_, err = p.rds().ModifyDBInstance(input)
	if err != nil {
		return "", fmt.Errorf("db error: %v", err)
	}

	return dbIdentifier, p.CreateDBSnapshot(app, resource, snapshot)
}

func (p *Provider) CreateDBSnapshot(app, resource, snapshot string) error {
	dbIdentifier, err := p.getResourceDBIdentifier(app, resource)
	if err != nil {
		return err
	}

	input := &rds.CreateDBSnapshotInput{
		DBSnapshotIdentifier: aws.String(snapshot),
		DBInstanceIdentifier: aws.String(dbIdentifier),
	}

	_, err = p.rds().CreateDBSnapshot(input)
	if err != nil {
		return fmt.Errorf("failed to create db snapshot: %v", err)
	}

	return nil
}

func (p *Provider) IsDBSnapshotComplete(snapshot string) (bool, error) {
	input := &rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(snapshot),
	}

	result, err := p.rds().DescribeDBSnapshots(input)
	if err != nil {
		return false, fmt.Errorf("failed to describe db snapshots: %v", err)
	}

	if len(result.DBSnapshots) == 0 {
		return false, fmt.Errorf("no snapshots found with identifier: %s", snapshot)
	}

	dbsnap := result.DBSnapshots[0]
	fmt.Printf("Current snapshot status: %s\n", aws.StringValue(dbsnap.Status))

	if aws.StringValue(dbsnap.Status) == "available" {
		fmt.Println("Snapshot is now available!")
		return true, nil
	}
	return false, nil
}

func (p *Provider) DeleteDB(resource string) error {
	input := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(resource),
		DeletionProtection:   aws.Bool(false),
	}

	_, err := p.rds().ModifyDBInstance(input)
	if err != nil {
		return fmt.Errorf("db error: %v", err)
	}

	_, err = p.rds().DeleteDBInstance(&rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:      &resource,
		FinalDBSnapshotIdentifier: aws.String(resource + "delete-snapshot"),
	})
	return err
}
