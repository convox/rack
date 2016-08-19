package aws_test

// func TestServiceCreateMysql(t *testing.T) {
//   aws, provider := StubAwsProvider(
//     cycleServiceDescribeStacksNotFound("convox-foo"),
//     cycleServiceDescribeStacksNotFound("foo"),
//     cycleServiceCreateStack("foo"),
//   )
//   defer aws.Close()

//   r, err := provider.ServiceCreate("foo", "mysql", nil)

//   assert.Nil(t, err)
//   assert.EqualValues(t, structs.Formation{
//     structs.ProcessFormation{
//       Balancer: "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com",
//       Name:     "web",
//       Count:    1,
//       Memory:   256,
//       CPU:      256,
//       Ports:    []int{80},
//     },
//   }, r)
// }

// func cycleServiceCreateStack(stack string) awsutil.Cycle {
//   return awsutil.Cycle{
//     awsutil.Request{"/", "", `Action=CreateStack&Capabilities.member.1=CAPABILITY_IAM&Parameters.member.1.ParameterKey=Password&Parameters.member.1.ParameterValue=9e60821344a787f471bc66612fe338&Parameters.member.2.ParameterKey=Subnets&Parameters.member.2.ParameterValue=&Parameters.member.3.ParameterKey=SubnetsPrivate&Parameters.member.3.ParameterValue=&Parameters.member.4.ParameterKey=Vpc&Parameters.member.4.ParameterValue=&Parameters.member.5.ParameterKey=VpcCidr&Parameters.member.5.ParameterValue=&StackName=foo&Tags.member.1.Key=Type&Tags.member.1.Value=service&Tags.member.2.Key=Rack&Tags.member.2.Value=convox&Tags.member.3.Key=Service&Tags.member.3.Value=mysql&Tags.member.4.Key=Name&Tags.member.4.Value=foo&Tags.member.5.Key=System&Tags.member.5.Value=convox&TemplateBody=%0A++%7B%0A++++%22AWSTemplateFormatVersion%22+%3A+%222010-09-09%22%2C%0A++++%22Conditions%22%3A+%7B%0A++++++%22Private%22%3A+%7B+%22Fn%3A%3AEquals%22%3A+%5B+%7B+%22Ref%22%3A+%22Private%22+%7D%2C+%22true%22+%5D+%7D%0A++++%7D%2C%0A++++%22Parameters%22%3A+%7B%0A++++++%22AllocatedStorage%22%3A+%7B%0A++++++++%22Type%22+%3A+%22Number%22%2C%0A++++++++%22Default%22+%3A+%2210%22%2C%0A++++++++%22Description%22+%3A+%22Allocated+storage+size+%28GB%29%22%0A++++++%7D%2C%0A++++++%22Database%22%3A+%7B%0A++++++++%22Type%22+%3A+%22String%22%2C%0A++++++++%22Default%22+%3A+%22app%22%2C%0A++++++++%22Description%22+%3A+%22Default+database+name%22%0A++++++%7D%2C%0A++++++%22InstanceType%22%3A+%7B%0A++++++++%22Type%22+%3A+%22String%22%2C%0A++++++++%22Default%22+%3A+%22db.t2.micro%22%2C%0A++++++++%22Description%22+%3A+%22Instance+class+for+database+nodes%22%0A++++++%7D%2C%0A++++++%22MultiAZ%22%3A+%7B%0A++++++++%22Type%22+%3A+%22String%22%2C%0A++++++++%22Default%22+%3A+%22false%22%2C%0A++++++++%22Description%22+%3A+%22Multiple+availability+zone%22%0A++++++%7D%2C%0A++++++%22Password%22%3A+%7B%0A++++++++%22Type%22+%3A+%22String%22%2C%0A++++++++%22Description%22+%3A+%22Server+password%22%0A++++++%7D%2C%0A++++++%22Private%22%3A+%7B%0A++++++++%22Type%22%3A+%22String%22%2C%0A++++++++%22Description%22%3A+%22Create+in+private+subnets%22%2C%0A++++++++%22Default%22%3A+%22false%22%2C%0A++++++++%22AllowedValues%22%3A+%5B+%22true%22%2C+%22false%22+%5D%0A++++++%7D%2C%0A++++++%22Subnets%22%3A+%7B%0A++++++++%22Type%22%3A+%22List%3CAWS%3A%3AEC2%3A%3ASubnet%3A%3AId%3E%22%2C%0A++++++++%22Description%22%3A+%22VPC+subnets%22%0A++++++%7D%2C%0A++++++%22SubnetsPrivate%22%3A+%7B%0A++++++++%22Type%22+%3A+%22List%3CAWS%3A%3AEC2%3A%3ASubnet%3A%3AId%3E%22%2C%0A++++++++%22Default%22+%3A+%22%22%2C%0A++++++++%22Description%22+%3A+%22VPC+private+subnets%22%0A++++++%7D%2C%0A++++++%22Username%22%3A+%7B%0A++++++++%22Type%22+%3A+%22String%22%2C%0A++++++++%22Default%22+%3A+%22app%22%2C%0A++++++++%22Description%22+%3A+%22Server+username%22%0A++++++%7D%2C%0A++++++%22Vpc%22%3A+%7B%0A++++++++%22Type%22%3A+%22AWS%3A%3AEC2%3A%3AVPC%3A%3AId%22%2C%0A++++++++%22Description%22%3A+%22VPC%22%0A++++++%7D%2C%0A++++++%22VpcCidr%22%3A+%7B%0A++++++++%22Description%22%3A+%22VPC+CIDR+Block%22%2C%0A++++++++%22Type%22%3A+%22String%22%0A++++++%7D%0A++++%7D%2C%0A++++%22Outputs%22%3A+%7B%0A++++++%22Port3306TcpAddr%22%3A+%7B+%22Value%22%3A+%7B+%22Fn%3A%3AGetAtt%22%3A+%5B+%22Instance%22%2C+%22Endpoint.Address%22+%5D+%7D+%7D%2C%0A++++++%22Port3306TcpPort%22%3A+%7B+%22Value%22%3A+%7B+%22Fn%3A%3AGetAtt%22%3A+%5B+%22Instance%22%2C+%22Endpoint.Port%22+%5D+%7D+%7D%2C%0A++++++%22EnvMysqlDatabase%22%3A+%7B+%22Value%22%3A+%7B+%22Ref%22%3A+%22Database%22+%7D+%7D%2C%0A++++++%22EnvMysqlPassword%22%3A+%7B+%22Value%22%3A+%7B+%22Ref%22%3A+%22Password%22+%7D+%7D%2C%0A++++++%22EnvMysqlUsername%22%3A+%7B+%22Value%22%3A+%7B+%22Ref%22%3A+%22Username%22+%7D+%7D%0A++++%7D%2C%0A++++%22Resources%22%3A+%7B%0A++++++%22SecurityGroup%22%3A+%7B%0A++++++++%22Type%22%3A+%22AWS%3A%3AEC2%3A%3ASecurityGroup%22%2C%0A++++++++%22Properties%22%3A+%7B%0A++++++++++%22GroupDescription%22%3A+%22mysql+service%22%2C%0A++++++++++%22SecurityGroupIngress%22%3A+%5B%0A++++++++++++%7B+%22IpProtocol%22%3A+%22tcp%22%2C+%22FromPort%22%3A+%223306%22%2C+%22ToPort%22%3A+%223306%22%2C+%22CidrIp%22%3A+%7B+%22Ref%22%3A+%22VpcCidr%22+%7D+%7D%0A++++++++++%5D%2C%0A++++++++++%22VpcId%22%3A+%7B+%22Ref%22%3A+%22Vpc%22+%7D%0A++++++++%7D%0A++++++%7D%2C%0A++++++%22SubnetGroup%22%3A+%7B%0A++++++++%22Type%22%3A+%22AWS%3A%3ARDS%3A%3ADBSubnetGroup%22%2C%0A++++++++%22Properties%22%3A+%7B%0A++++++++++%22DBSubnetGroupDescription%22%3A+%22mysql+service%22%2C%0A++++++++++%22SubnetIds%22%3A+%7B+%22Fn%3A%3AIf%22%3A+%5B+%22Private%22%2C%0A++++++++++++%7B+%22Ref%22%3A+%22SubnetsPrivate%22+%7D%2C%0A++++++++++++%7B+%22Ref%22%3A+%22Subnets%22+%7D%0A++++++++++%5D+%7D%0A++++++++%7D%0A++++++%7D%2C%0A++++++%22Instance%22%3A+%7B%0A++++++++%22Type%22%3A+%22AWS%3A%3ARDS%3A%3ADBInstance%22%2C%0A++++++++%22Properties%22%3A+%7B%0A++++++++++%22AllocatedStorage%22%3A+%7B+%22Ref%22%3A+%22AllocatedStorage%22+%7D%2C%0A++++++++++%22DBInstanceClass%22%3A+%7B+%22Ref%22%3A+%22InstanceType%22+%7D%2C%0A++++++++++%22DBInstanceIdentifier%22%3A+%7B+%22Ref%22%3A+%22AWS%3A%3AStackName%22+%7D%2C%0A++++++++++%22DBName%22%3A+%7B+%22Ref%22%3A+%22Database%22+%7D%2C%0A++++++++++%22DBSubnetGroupName%22%3A+%7B+%22Ref%22%3A+%22SubnetGroup%22+%7D%2C%0A++++++++++%22Engine%22%3A+%22mysql%22%2C%0A++++++++++%22EngineVersion%22%3A+%225.6.27%22%2C%0A++++++++++%22MasterUsername%22%3A+%7B+%22Ref%22%3A+%22Username%22+%7D%2C%0A++++++++++%22MasterUserPassword%22%3A+%7B+%22Ref%22%3A+%22Password%22+%7D%2C%0A++++++++++%22MultiAZ%22%3A+%7B+%22Ref%22%3A+%22MultiAZ%22+%7D%2C%0A++++++++++%22Port%22%3A+%223306%22%2C%0A++++++++++%22PubliclyAccessible%22%3A+%22false%22%2C%0A++++++++++%22StorageType%22%3A+%22gp2%22%2C%0A++++++++++%22VPCSecurityGroups%22%3A+%5B+%7B+%22Ref%22%3A+%22SecurityGroup%22+%7D+%5D%0A++++++++%7D%0A++++++%7D%0A++++%7D%0A++%7D%0A&Version=2010-05-15`},
//     awsutil.Response{400, `
//     <ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
//       <Error>
//         <Type>Sender</Type>
//         <Code>ValidationError</Code>
//         <Message>Stack with id ` + stack + ` does not exist</Message>
//       </Error>
//       <RequestId>bc91dc86-5803-11e5-a24f-85fde26a90fa</RequestId>
//     </ErrorResponse>
//   `},
//   }
// }

// func cycleServiceDescribeStacksNotFound(stack string) awsutil.Cycle {
//   return awsutil.Cycle{
//     awsutil.Request{"/", "", `Action=DescribeStacks&StackName=` + stack + `&Version=2010-05-15`},
//     awsutil.Response{400, `
//     <ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
//       <Error>
//         <Type>Sender</Type>
//         <Code>ValidationError</Code>
//         <Message>Stack with id ` + stack + ` does not exist</Message>
//       </Error>
//       <RequestId>bc91dc86-5803-11e5-a24f-85fde26a90fa</RequestId>
//     </ErrorResponse>
//   `},
//   }
// }

// var serviceDescribeStacksExistsCycle = awsutil.Cycle{
//   awsutil.Request{"/", "", `Action=DescribeStacks&StackName=convox-foo&Version=2010-05-15`},
//   awsutil.Response{200, `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
//   <DescribeStacksResult>
//     <Stacks>
//       <member>
//         <Tags>
//           <member>
//             <Value>httpd</Value>
//             <Key>Name</Key>
//           </member>
//           <member>
//             <Value>app</Value>
//             <Key>Type</Key>
//           </member>
//           <member>
//             <Value>convox</Value>
//             <Key>System</Key>
//           </member>
//           <member>
//             <Value>convox</Value>
//             <Key>Rack</Key>
//           </member>
//         </Tags>
//         <StackId>arn:aws:cloudformation:us-east-1:132866487567:stack/convox-httpd/53df3c30-f763-11e5-bd5d-50d5cd148236</StackId>
//         <StackStatus>UPDATE_COMPLETE</StackStatus>
//         <StackName>convox-httpd</StackName>
//         <LastUpdatedTime>2016-03-31T17:12:16.275Z</LastUpdatedTime>
//         <NotificationARNs/>
//         <CreationTime>2016-03-31T17:09:28.583Z</CreationTime>
//         <Parameters>
//           <member>
//             <ParameterValue>https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RVFETUHHKKD/env</ParameterValue>
//             <ParameterKey>Environment</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue/>
//             <ParameterKey>WebPort80Certificate</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>No</ParameterValue>
//             <ParameterKey>WebPort80ProxyProtocol</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>256</ParameterValue>
//             <ParameterKey>WebCpu</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>256</ParameterValue>
//             <ParameterKey>WebMemory</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920</ParameterValue>
//             <ParameterKey>Key</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue/>
//             <ParameterKey>Repository</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>80</ParameterValue>
//             <ParameterKey>WebPort80Balancer</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>56694</ParameterValue>
//             <ParameterKey>WebPort80Host</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>vpc-f8006b9c</ParameterValue>
//             <ParameterKey>VPC</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>1</ParameterValue>
//             <ParameterKey>WebDesiredCount</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>convox-Cluster-1E4XJ0PQWNAYS</ParameterValue>
//             <ParameterKey>Cluster</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>subnet-d4e85cfe,subnet-103d5a66,subnet-57952a0f</ParameterValue>
//             <ParameterKey>SubnetsPrivate</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>RVFETUHHKKD</ParameterValue>
//             <ParameterKey>Release</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>No</ParameterValue>
//             <ParameterKey>WebPort80Secure</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>subnet-13de3139,subnet-b5578fc3,subnet-21c13379</ParameterValue>
//             <ParameterKey>Subnets</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>20160330143438-command-exec-form</ParameterValue>
//             <ParameterKey>Version</ParameterKey>
//           </member>
//           <member>
//             <ParameterValue>Yes</ParameterValue>
//             <ParameterKey>Private</ParameterKey>
//           </member>
//         </Parameters>
//         <DisableRollback>false</DisableRollback>
//         <Capabilities>
//           <member>CAPABILITY_IAM</member>
//         </Capabilities>
//         <Outputs>
//           <member>
//             <OutputValue>httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com</OutputValue>
//             <OutputKey>BalancerWebHost</OutputKey>
//           </member>
//           <member>
//             <OutputValue>convox-httpd-Kinesis-1MAP0GJ6RITJF</OutputValue>
//             <OutputKey>Kinesis</OutputKey>
//           </member>
//           <member>
//             <OutputValue>convox-httpd-LogGroup-L4V203L35WRM</OutputValue>
//             <OutputKey>LogGroup</OutputKey>
//           </member>
//           <member>
//             <OutputValue>132866487567</OutputValue>
//             <OutputKey>RegistryId</OutputKey>
//           </member>
//           <member>
//             <OutputValue>convox-httpd-hqvvfosgxt</OutputValue>
//             <OutputKey>RegistryRepository</OutputKey>
//           </member>
//           <member>
//             <OutputValue>convox-httpd-settings-139bidzalmbtu</OutputValue>
//             <OutputKey>Settings</OutputKey>
//           </member>
//           <member>
//             <OutputValue>80</OutputValue>
//             <OutputKey>WebPort80Balancer</OutputKey>
//           </member>
//           <member>
//             <OutputValue>httpd-web-7E5UPCM</OutputValue>
//             <OutputKey>WebPort80BalancerName</OutputKey>
//           </member>
//         </Outputs>
//       </member>
//     </Stacks>
//   </DescribeStacksResult>
//   <ResponseMetadata>
//     <RequestId>d5220387-f76d-11e5-912c-531803b112a4</RequestId>
//   </ResponseMetadata>
// </DescribeStacksResponse>`},
// }
