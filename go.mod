module github.com/taylormonacelli/smoggytexas

go 1.21.1

require (
	github.com/aws/aws-sdk-go-v2 v1.32.2
	github.com/aws/aws-sdk-go-v2/config v1.28.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.184.0
	github.com/dustin/go-humanize v1.0.1
	github.com/taylormonacelli/lemondrop v0.0.20
)

require (
	github.com/adrg/xdg v0.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.41 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.43 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.38.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.23.0 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/taylormonacelli/lemondrop => ../lemondrop
