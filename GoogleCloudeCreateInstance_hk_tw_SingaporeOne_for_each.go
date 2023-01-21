package main

//私钥密码：notasecret
//私钥路径：/home/rowe/下载/stunning-grin-374606-fcf5cfffe476.p12
import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"strconv"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

// createWithLocalSSD creates a new VM instance with Debian 10 operating system and a local SSD attached.
func createWithLocalSSD(projectID, zone, instanceName, machineType, sourceImage, networkName, accountadc_config string) {
	// projectID := "your_project_id"
	// zone := "europe-central2-b"
	// instanceName := "your_instance_name"
	// machineType := "n1-standard-1","local-ssd"
	// sourceImage := "projects/debian-cloud/global/images/family/debian-10"
	// networkName := "global/networks/default"

	ctx := context.Background()

	instancesClient, err := compute.NewInstancesRESTClient(ctx, option.WithCredentialsFile(accountadc_config))
	if err != nil {
		fmt.Printf("NewInstancesRESTClient: %s\n", err)
	}
	defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
		Project: projectID,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name: proto.String(instanceName),
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(512),
						SourceImage: proto.String(sourceImage),
						DiskType:    proto.String(fmt.Sprintf("zones/%s/diskTypes/pd-standard", zone)), //pd-ssd
					},
					AutoDelete: proto.Bool(true),
					Boot:       proto.Bool(true),
					Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
				},
			},
			CanIpForward: proto.Bool(true),
			MachineType:  proto.String(fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType)),
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name: proto.String(networkName),
					AccessConfigs: []*computepb.AccessConfig{
						{
							Type: proto.String("ONE_TO_ONE_NAT"),
							Name: proto.String("External NAT"),
						},
					},
				},
			},
		},
	}

	op, err := instancesClient.Insert(ctx, req)
	if err != nil {
		fmt.Printf("unable to create instance: %s\n", err)
	}
	if op != nil {
		if err = op.Wait(ctx); err != nil {
			fmt.Printf("unable to wait for the operation: %s\n", err)
		}
	}
}

func listInstances(projectID, zone string, accountadc_config string) {
	// projectID := "your_project_id"
	// zone := "europe-central2-b"
	ctx := context.Background()
	instancesClient, err := compute.NewInstancesRESTClient(ctx, option.WithCredentialsFile(accountadc_config))
	if err != nil {
		fmt.Errorf("NewInstancesRESTClient: %s\n", err)
	}
	defer instancesClient.Close()

	req := &computepb.ListInstancesRequest{
		Project: projectID,
		Zone:    zone,
	}

	it := instancesClient.List(ctx, req)
	if it == nil {
		fmt.Printf("Instances found in zone %s:\n", zone)
	}
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("NewInstancesRESTClient: %s\n", err)
		}
		InstanceIPv4 := instance.GetNetworkInterfaces()[0].AccessConfigs[0].NatIP
		fmt.Printf("%s %s\n", instance.GetName(), *InstanceIPv4)
	}
}

// createFirewallRule creates a firewall rule allowing for incoming HTTP and HTTPS access from the entire Internet.
func createFirewallRule(projectID, firewallRuleName, networkName string, accountadc_config string) {
	// projectID := "your_project_id"
	// firewallRuleName := "europe-central2-b"
	// networkName := "global/networks/default"

	ctx := context.Background()
	firewallsClient, err := compute.NewFirewallsRESTClient(ctx, option.WithCredentialsFile(accountadc_config))
	if err != nil {
		fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer firewallsClient.Close()

	firewallRule := &computepb.Firewall{
		Allowed: []*computepb.Allowed{
			{
				IPProtocol: proto.String("all"),
				Ports:      []string{},
			},
		},
		Direction:   proto.String(computepb.Firewall_INGRESS.String()),
		Name:        &firewallRuleName,
		TargetTags:  []string{},
		Network:     &networkName,
		Description: proto.String("Allowing TCP traffic on port all from Internet."),
	}

	// Note that the default value of priority for the firewall API is 1000.
	// If you check the value of `firewallRule.GetPriority()` at this point it
	// will be equal to 0, however it is not treated as "set" by the library and thus
	// the default will be applied to the new rule. If you want to create a rule that
	// has priority == 0, you need to explicitly set it so:

	firewallRule.Priority = proto.Int32(1000)

	req := &computepb.InsertFirewallRequest{
		Project:          projectID,
		FirewallResource: firewallRule,
	}

	op, err := firewallsClient.Insert(ctx, req)
	if err != nil {
		fmt.Printf("unable to create firewall rule: %v", err)
	}
	if op != nil {
		if err = op.Wait(ctx); err != nil {
			fmt.Printf("unable to wait for the operation: %v", err)
		}
	}

	fmt.Printf("Firewall rule created\n")

}

func SetCommonInstanceMetadata(projectID string, instanceName string, zone string, accountadc_config string) error {
	ctx := context.Background()
	Client, err := compute.NewProjectsRESTClient(ctx, option.WithCredentialsFile(accountadc_config))
	if err != nil {
		fmt.Errorf("Client Error: %v", err)
	}

	defer Client.Close()

	//req := &computepb.NewInstancesRESTClient{
	//	Project: projectID,
	//	Zone: zone,
	//}
	//metadata_req,err:= Client.Get(ctx,req)
	//
	//if err !=nil {
	//	fmt.Printf("Get Error: %s\n", err)
	//}
	//fingerprint := metadata_req
	setmeta_req := &computepb.SetCommonInstanceMetadataProjectRequest{
		Project: projectID,
		//Instance: instanceName,
		//Zone: zone,
		MetadataResource: &computepb.Metadata{
			Items: []*computepb.Items{
				{
					Key:   proto.String("ssh-keys"),
					Value: proto.String("gcp:ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDMgcoADSldk0Kxmp/aB1zxFmzEX+eBUzu+9uuTuNIeoPndtkI21mpEWKt+gnXctMxqLygFda2ZHagQLFhLYppFPPt1jYcefcf9BWRXy79h2DwoLV/sDRqeCGgkN5KTSmioU6MvkEOpx71bGN9M5euQgMWxWLywmup+7+Qp/4X7wMvtlx7PvNkCGMikOnD+tBi4CtaeovB+Cm+9BKWy0uIvUcFj6lCh+/yRFyP7/HA9n2JtnSNbFJcR3q5UJGA2jqGyUxcgRcNnlTkvxDZUOyeXHsjXULQ0D4FQStKnRToRJ7C9jfG56aEvNTx7Bcu0+4+LMG/onFXM9vd6lvpKTuyFapW7eeU+QmB5F9niAV1zzfciE0ipJx6WKUaBZRbthosP41uDFJ4LoQO2brEnF/9K6jDXtgyddsCyFUy5z1E4S4ddcBTOAARrjE2aOygfV5XdEpeXBDvxNql42hN8mNTE8oN4/RgImPIHyunREHgp36coTsQUUqCeV3sDmimIVUc="),
				},
			},
		},
		//Fingerprint: proto.String(fingerprint),
	}

	op, err := Client.SetCommonInstanceMetadata(ctx, setmeta_req)
	if err != nil {
		fmt.Printf("Set Metadata Error: %v", err)
	}
	if op != nil {
		if err = op.Wait(ctx); err != nil {
			fmt.Printf("unable to wait for the operation: %s\n", err)
			return err
		}
	}
	return nil
}

func main() {
	var projectID string
	var zone string
	var instanceName string
	var machineType string
	var sourceImage string
	var networkName string
	var Firewald int
	var instanceNum int
	var accountadc_credentials_config string
	flag.StringVar(&projectID, "p", "", "项目ID，必须，如：stunning-grin-374606")
	flag.StringVar(&zone, "z", "asia-east2-b", "区域,默认：此项无效，绝对——hk-c,tw-c,Singapore-b,但在3个以内可以使用-N 3这个个参数一起批量申请3个")
	flag.StringVar(&instanceName, "i", "instance", "实例名称，默认：instances+数字序列")
	flag.StringVar(&machineType, "t", "n2-standard-4", "实例类型,默认：n2-standard-4(4c16g)")
	flag.StringVar(&sourceImage, "o", "projects/centos-cloud/global/images/centos-7-v20221206", "镜像名称，系统镜像名,默认：projects/centos-cloud/global/images/centos-7-v20221206") // projects/centos-cloud/global/images/centos-7-v20221206")
	flag.StringVar(&networkName, "n", "projects/"+projectID+"/global/networks/default", "网络名称,默认：global/+项目ID+/networks/default")
	flag.IntVar(&instanceNum, "N", 3, "申请实例数量,默认：3")
	flag.StringVar(&accountadc_credentials_config, "c", "", "网络名称,默认：空，gcloud auth application-default login，执行后会打印在屏幕“application_default_credentials.json文件路径”，参数填写该文件路径。\n"+
		"Linux、macOS：$HOME/.config/gcloud/application_default_credentials.json\nWindows：%APPDATA%\\gcloud\\application_default_credentials.json")
	flag.IntVar(&Firewald, "f", 0, "，防火墙允许所有路由添加，默认：0，不执行添加路由，虚手动设置参数：-f 1 ，才执行添加防火墙路由规则")
	flag.Parse()
	//test_run:="stunning-grin-374606"
	//projectID = test_run

	NetworkName := "projects/" + projectID + "/global/networks/default"
	if Firewald == 1 {
		createFirewallRule(projectID, "default-allow-all", NetworkName, accountadc_credentials_config)
	}
	for i := 1; i <= instanceNum; i++ {
		if projectID != "" {
			time.Sleep(time.Second * 10)
			if i <= 2 {
				zone = "asia-east" + strconv.Itoa(i) + "-c"
				InstanceName := zone + "-" + instanceName + "-" + strconv.Itoa(i)
				createWithLocalSSD(projectID, zone, InstanceName, machineType, sourceImage, NetworkName, accountadc_credentials_config)
			} else {
				zone = "asia-southeast1-b"
				InstanceName := zone + "-" + instanceName + "-" + strconv.Itoa(i)
				createWithLocalSSD(projectID, zone, InstanceName, machineType, sourceImage, NetworkName, accountadc_credentials_config)
			}
		} else {
			fmt.Printf("使用的--help查看参数具体要求\n")
			fmt.Printf("如果是初次使用请安装Google Cloud CLI,本程序严重依赖该组件运行，如无该组件，运行无效果。\nWindows安装包下载地址：\n" +
				"https://dl.google.com/dl/cloudsdk/channels/rapid/GoogleCloudSDKInstaller.exe\n" + "Liunx安装下载地址：\n" +
				"https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-412.0.0-linux-x86_64.tar.gz\n" + "Mac下载地址：\n" +
				"https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-412.0.0-darwin-x86_64.tar.gz?hl=zh-cn\n\n")
			fmt.Printf("启动安装程序并按提示操作。安装程序已经过 Google LLC 签名。如果您使用的是屏幕阅读器，请选中启用屏幕阅读器模式复选框(linux\\mac安装执行时添加参数：--screen-reader=true\n" +
				")。此选项会将 gcloud 配置为使用状态跟踪器而不是 Unicode 旋转图标，以百分比表示显示进度和展开的表。如需了解详情，请参阅无障碍功能指南。Cloud SDK 要求安装 Python；受支持的版本是 \n" +
				"Python 3（3.5 到 3.9）。默认情况下，Windows 版本的 Cloud SDK 捆绑了Python 3。如需使用 Cloud SDK，您的操作系统必须能够运行受支持的 Python 版本。安装程序会安装所有必需的依赖项\n" +
				"（包括所需的 Python 版本）。虽然 Cloud SDK 默认安装和管理 Python 3，但您可以根据需要使用已安装的 Python，只需取消选中“安装捆绑的 Python”选项即可。请参阅 、gcloud topic startup，\n" +
				"了解如何使用现有 Python 安装。安装完成后，安装程序会为您提供创建开始菜单和桌面快捷方式、启动 Google Cloud CLI shell 以及配置 gcloud CLI 的选项。确保已选择用于启动 shell 并配置安\n" +
				"装的选项。安装程序会启动终端窗口并运行 gcloud init 命令。默认安装不包括使用 gcloud 命令部署应用所必需的 App Engine 扩展程序。您可以使用 gcloud CLI 组件管理器安装这些组件。\n\n")
			fmt.Printf("本程序由golang编写，安装以上组件后，Linux、Mac安装后，还需安装：”app-engine-go“组件，执行：”gcloud components install app-engine-go\n\n")
			fmt.Printf("上述操作完成以后，使用命令：gcloud auth application-default login ，弹出浏览器登陆googlecloud帐号，生成配置文件，\n" +
				"如有多帐号，请使用：gcloud auth application-default revoke 退出登陆，再登陆新帐号。并拷贝更名存档配置文件，后续程序执行，直接使用的-c 参数更换认证账户配置文件路径即可。\n")
			fmt.Printf("本程序自动生成ssh-keys到项目下所有实例主机，账户名：gcp,key请找开发者索取！！！！！！！！！！！！！！！！！")
			break
		}

	}
	SetCommonInstanceMetadata(projectID, instanceName, zone, accountadc_credentials_config)
	if projectID != "" {
		fmt.Printf("项目 = " + projectID + "\n")
		//fmt.Printf("第"+strconv.Itoa(i)+"个实例申请操作完成\n")
		//fmt.Printf("等待2秒钟，获取项目创建的实例\n")
		time.Sleep(time.Second * 2)
		listInstances(projectID, "asia-east1-c", accountadc_credentials_config)
		listInstances(projectID, "asia-east2-c", accountadc_credentials_config)
		listInstances(projectID, "asia-southeast1-b", accountadc_credentials_config)
	} else {
		fmt.Printf("使用的--help查看参数具体要求\n")
	}

}

//CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build GoogleCloudeCreateInstance.go
