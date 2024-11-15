'use client';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useEffect, useState } from "react";

import Link from 'next/link';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import { useMutation } from "@apollo/client";
import { VerifyAccountIdMutation } from "@/gql/graphql";
import { VERIFY_ACCOUNT_ID } from "./gql";

function isValidAwsAccountId(accountId: string): boolean {
  // Check if the accountId matches the pattern of exactly 12 digits
  const regex = /^\d{12}$/;
  return regex.test(accountId);
}

const Installation = ({ externalId, refetch }: { externalId: string, refetch: () => Promise<unknown> }) => {
  const [installMethod, setInstallMethod] = useState('cloudformation');
  const [accountId, setAccountId] = useState('');

  const [verifyAccountId, { data, loading, error }] = useMutation<VerifyAccountIdMutation>(VERIFY_ACCOUNT_ID, { variables: { accountId } });

  useEffect(() => {
    if (data?.verifyAccountId) refetch();
  }, [data]);

  const renderInstructions = () => {
    if (process.env.NEXT_PUBLIC_SELF_HOSTING) {
      return (
        <div className="flex flex-col gap-5">
          <p>
            To get started with Guard in self-hosting mode, you need to create an IAM User with the required permissions. Follow these steps:
          </p>
          <div className="flex flex-col gap-2">
            <p>1. Go to the AWS IAM Console and create a new IAM User with programmatic access.</p>
            <p>2. Attach the policy document provided in the Guard repository:</p>
            <div className="font-mono text-sm p-2 bg-neutral-100 dark:bg-neutral-900 rounded">
              <Link target="_blank" href="https://github.com/guard-dev/guard/blob/main/templates/guard-self-host-policy.json">
                guard-self-host-policy.json
              </Link>
            </div>
            <p>3. After creating the IAM User, securely store the Access Key ID and Secret Access Key.</p>
            <p>4. Enter your 12-digit AWS Account ID below to verify the connection and complete onboarding.</p>
            <div className="flex flex-row items-center justify-between gap-2">
              <Input placeholder="12 Digit AWS Account ID" value={accountId} onChange={e => setAccountId(e.target.value)} />
              <Button disabled={!isValidAwsAccountId(accountId) || loading} className="rounded-lg" onClick={() => verifyAccountId()}>
                Verify Connection
              </Button>
            </div>
          </div>
          {data?.verifyAccountId === true && <div>Verified. Redirecting...</div>}
          {error && <div>{error.message}</div>}
        </div>
      );
    } else {
      switch (installMethod) {
        case 'cloudformation':
          return (
            <div className="flex flex-col gap-5">
              <p>
                Guard uses <span className="underline"><Link href={"https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html"} target="_blank">AWS Security Token Service (STS)</Link></span> to request temporary, limited-privilege credentials to access your AWS account securely.
              </p>
              <p>
                Follow these steps to set up CloudFormation:
              </p>
              <div className="flex flex-col gap-2">
                <p>
                  {`1. Click the button below to deploy the CloudFormation stack, which will create the required IAM role with read-only permissions.`}
                </p>
                <div className="flex flex-row items-center justify-start">
                  <Button className="w-max rounded-lg" variant={"secondary"} disabled={externalId === ""}>
                    <Link href={`https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://guard-cloudformation-templates.s3.amazonaws.com/guard-scan-role.yaml&stackName=GuardSecurityScanRole&param_ExternalId=${externalId}`} target="_blank">
                      Launch CloudFormation Stack
                    </Link>
                  </Button>
                </div>
              </div>
              <div className="flex flex-col gap-2">
                <p>2. Enter your 12-digit AWS Account ID below to verify the connection and complete onboarding.</p>
                <div className="flex flex-row items-center justify-between gap-2">
                  <Input placeholder="12 Digit AWS Account ID" value={accountId} onChange={e => setAccountId(e.target.value)} />
                  <Button disabled={!isValidAwsAccountId(accountId) || loading} className="rounded-lg" onClick={() => verifyAccountId()}>
                    Verify Connection
                  </Button>
                </div>
              </div>
              {data?.verifyAccountId === true && <div>Verified. Redirecting...</div>}
              {error && <div>Could not verify account. Please make sure that the CloudFormation stack is created and the account ID is correct.</div>}
            </div>
          );
        default:
          return null;
      }
    }
  }

  return (
    <div className="flex flex-col rounded-lg gap-2">
      <div className="flex flex-row justify-between text-lg items-center border-b pb-3">
        <div>Choose installation method</div>
        {!process.env.NEXT_PUBLIC_SELF_HOSTING && (
          <Select value={installMethod} onValueChange={(e) => setInstallMethod(e)}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Installation Method" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="cloudformation">CloudFormation</SelectItem>
            </SelectContent>
          </Select>
        )}
      </div>
      <div className="pt-1">
        {renderInstructions()}
      </div>
    </div>
  );
}

export default Installation;

