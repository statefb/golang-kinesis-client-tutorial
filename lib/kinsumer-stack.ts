import * as cdk from "aws-cdk-lib";
import { Construct } from "constructs";
import * as kinesis from "aws-cdk-lib/aws-kinesis";
import * as iam from "aws-cdk-lib/aws-iam";
import * as dynamodb from "aws-cdk-lib/aws-dynamodb";
import * as ec2 from "aws-cdk-lib/aws-ec2";
import * as ecs from "aws-cdk-lib/aws-ecs";
import * as logs from "aws-cdk-lib/aws-logs";
import * as ecsPatterns from "aws-cdk-lib/aws-ecs-patterns";
import * as path from "path";

export class KinsumerStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
    const vpc = new ec2.Vpc(this, "VPC", {});
    const cluster = new ecs.Cluster(this, "Cluster", { vpc });

    const taskDefinition = new ecs.FargateTaskDefinition(
      this,
      "TaskDefinition",
      {
        // ref: https://docs.aws.amazon.com/ja_jp/AmazonECS/latest/developerguide/task-cpu-memory-error.html
        cpu: 1024,
        memoryLimitMiB: 2048,
      }
    );

    // KDS
    const stream = new kinesis.Stream(this, "Stream", {
      streamMode: kinesis.StreamMode.PROVISIONED,
      shardCount: 8,
    });
    stream.grantRead(taskDefinition.taskRole);

    const logGroup = new logs.LogGroup(this, "TaskLogGroup", {
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      retention: logs.RetentionDays.ONE_WEEK,
    });
    const container = taskDefinition.addContainer("Container", {
      image: ecs.ContainerImage.fromAsset(path.join(__dirname, "../consumer")),
      logging: ecs.LogDriver.awsLogs({
        streamPrefix: "kcl-consumer",
        logGroup: logGroup,
      }),
      // 環境変数
      environment: {
        STREAM_NAME: stream.streamName,
        // スタック名。kinsumerのインスタンス時に名前を付与するために使用
        STACK_NAME: this.stackName,
        // health check用のポート
        PORT: "8080",
      },
    });
    container.addPortMappings({ containerPort: 8080 });

    // ECS Service
    const service = new ecsPatterns.ApplicationLoadBalancedFargateService(
      this,
      "Service",
      {
        cluster,
        taskDefinition,
        desiredCount: 4,
      }
    );
    service.targetGroup.configureHealthCheck({
      path: "/health",
    });

    // オートスケーリング設定
    // 増減時はkinsumerがハンドリングしてくれるが、テスト時はタスク数を固定しておく方が便利のため一旦コメントアウト
    const scaling = service.service.autoScaleTaskCount({
      minCapacity: 1,
      maxCapacity: 10,
    });
    // scaling.scaleOnCpuUtilization("CpuScaling", {
    //   targetUtilizationPercent: 70,
    //   scaleInCooldown: cdk.Duration.seconds(60),
    //   scaleOutCooldown: cdk.Duration.seconds(60),
    // });

    // DDB
    // kinsumerのアプリ名 (今回はスタック名) + 特定の名前のテーブルが必要
    const clientTable = new dynamodb.Table(this, "ClientTable", {
      tableName: `${this.stackName}_clients`,
      partitionKey: { name: "ID", type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });
    const checkpointTable = new dynamodb.Table(this, "CheckpointTable", {
      tableName: `${this.stackName}_checkpoints`,
      partitionKey: { name: "Shard", type: dynamodb.AttributeType.STRING },
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });
    const metaTable = new dynamodb.Table(this, "MetaTable", {
      tableName: `${this.stackName}_metadata`,
      partitionKey: { name: "Key", type: dynamodb.AttributeType.STRING },
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });
    clientTable.grantReadWriteData(taskDefinition.taskRole);
    checkpointTable.grantReadWriteData(taskDefinition.taskRole);
    metaTable.grantReadWriteData(taskDefinition.taskRole);
    taskDefinition.taskRole.addManagedPolicy(
      iam.ManagedPolicy.fromAwsManagedPolicyName("AmazonDynamoDBFullAccess")
    );
  }
}
