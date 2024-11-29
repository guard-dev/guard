'use client';

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
  const [accountId, setAccountId] = useState('');
  const [verifyAccountId, { data, loading, error }] = useMutation<VerifyAccountIdMutation>(VERIFY_ACCOUNT_ID, { variables: { accountId } });

  useEffect(() => {
    if (data?.verifyAccountId) refetch();
  }, [data]);

  const renderSelfHostingInstructions = () => (
    <div className="space-y-8">
      <p className="text-lg text-muted-foreground">
        To get started with Guard in self-hosting mode, you need to create an IAM User with the required permissions. Follow these steps:
      </p>

      <div className="space-y-6">
        {/* Step 1 */}
        <div className="relative pl-8">
          <div className="absolute left-0 top-1.5 w-4 h-4 rounded-full bg-gradient-to-br from-neutral-800 to-neutral-900 dark:from-neutral-200 dark:to-white flex items-center justify-center">
            <span className="text-xs text-white dark:text-neutral-900 font-bold">1</span>
          </div>
          <div className="space-y-2">
            <h3 className="font-[family-name:var(--font-geist-mono)] text-base">Create IAM User</h3>
            <p className="text-sm text-muted-foreground">Go to the AWS IAM Console and create a new IAM User with programmatic access.</p>
          </div>
        </div>

        {/* Step 2 */}
        <div className="relative pl-8">
          <div className="absolute left-0 top-1.5 w-4 h-4 rounded-full bg-gradient-to-br from-neutral-800 to-neutral-900 dark:from-neutral-200 dark:to-white flex items-center justify-center">
            <span className="text-xs text-white dark:text-neutral-900 font-bold">2</span>
          </div>
          <div className="space-y-2">
            <h3 className="font-[family-name:var(--font-geist-mono)] text-base">Attach Policy</h3>
            <p className="text-sm text-muted-foreground">Attach the policy document provided in the Guard repository:</p>
            <div className="mt-2 p-3 rounded-lg bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 border">
              <Link
                target="_blank"
                href="https://github.com/guard-dev/guard/blob/main/templates/guard-self-host-policy.json"
                className="text-sm font-[family-name:var(--font-geist-mono)] hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
              >
                guard-self-host-policy.json
              </Link>
            </div>
          </div>
        </div>

        {/* Step 3 */}
        <div className="relative pl-8">
          <div className="absolute left-0 top-1.5 w-4 h-4 rounded-full bg-gradient-to-br from-neutral-800 to-neutral-900 dark:from-neutral-200 dark:to-white flex items-center justify-center">
            <span className="text-xs text-white dark:text-neutral-900 font-bold">3</span>
          </div>
          <div className="space-y-2">
            <h3 className="font-[family-name:var(--font-geist-mono)] text-base">Store Credentials</h3>
            <p className="text-sm text-muted-foreground">After creating the IAM User, securely store the Access Key ID and Secret Access Key.</p>
          </div>
        </div>

        {/* Step 4 */}
        <div className="relative pl-8">
          <div className="absolute left-0 top-1.5 w-4 h-4 rounded-full bg-gradient-to-br from-neutral-800 to-neutral-900 dark:from-neutral-200 dark:to-white flex items-center justify-center">
            <span className="text-xs text-white dark:text-neutral-900 font-bold">4</span>
          </div>
          <div className="space-y-2">
            <h3 className="font-[family-name:var(--font-geist-mono)] text-base">Verify Connection</h3>
            <p className="text-sm text-muted-foreground">Enter your 12-digit AWS Account ID below to verify the connection and complete onboarding.</p>
            <div className="flex items-center gap-4 mt-4">
              <Input
                placeholder="12 Digit AWS Account ID"
                value={accountId}
                onChange={e => setAccountId(e.target.value)}
                className="font-[family-name:var(--font-geist-mono)]"
              />
              <Button
                disabled={!isValidAwsAccountId(accountId) || loading}
                onClick={() => verifyAccountId()}
                className="rounded-full bg-gradient-to-r from-neutral-900 via-neutral-800 to-neutral-700 hover:from-neutral-800 hover:via-neutral-700 hover:to-neutral-600 text-white border-0 dark:from-white dark:via-neutral-200 dark:to-neutral-400 dark:hover:from-neutral-200 dark:hover:via-neutral-300 dark:hover:to-neutral-500 dark:text-neutral-900"
              >
                {loading ? 'Verifying...' : 'Verify Connection'}
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Status Messages */}
      {data?.verifyAccountId === true && (
        <div className="mt-6 p-4 rounded-lg bg-gradient-to-r from-green-100 to-green-50 dark:from-green-900 dark:to-green-800 text-green-800 dark:text-green-100 border border-green-200 dark:border-green-700">
          Connection verified successfully! Redirecting...
        </div>
      )}
      {error && (
        <div className="mt-6 p-4 rounded-lg bg-gradient-to-r from-red-100 to-red-50 dark:from-red-900 dark:to-red-800 text-red-800 dark:text-red-100 border border-red-200 dark:border-red-700">
          {error.message}
        </div>
      )}
    </div>
  );

  const renderCloudFormationInstructions = () => (
    <div className="space-y-8">
      <div className="p-4 rounded-lg bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 border">
        <p className="text-sm">
          Guard uses{" "}
          <Link
            href="https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html"
            target="_blank"
            className="text-blue-600 dark:text-blue-400 hover:underline"
          >
            AWS Security Token Service (STS)
          </Link>
          {" "}to request temporary, limited-privilege credentials to access your AWS account securely.
        </p>
      </div>

      <div className="space-y-6">
        {/* Step 1 */}
        <div className="relative pl-8">
          <div className="absolute left-0 top-1.5 w-4 h-4 rounded-full bg-gradient-to-br from-neutral-800 to-neutral-900 dark:from-neutral-200 dark:to-white flex items-center justify-center">
            <span className="text-xs text-white dark:text-neutral-900 font-bold">1</span>
          </div>
          <div className="space-y-2">
            <h3 className="font-[family-name:var(--font-geist-mono)] text-base">Deploy CloudFormation Stack</h3>
            <p className="text-sm text-muted-foreground">
              Click the button below to deploy the CloudFormation stack, which will create the required IAM role with read-only permissions.
            </p>
            <Button
              variant="secondary"
              disabled={externalId === ""}
              className="mt-2 rounded-full bg-gradient-to-r from-neutral-100 via-white to-neutral-200 hover:from-white hover:via-neutral-100 hover:to-neutral-300 dark:from-neutral-800 dark:via-neutral-900 dark:to-neutral-950 dark:hover:from-neutral-700 dark:hover:via-neutral-800 dark:hover:to-neutral-900 border border-neutral-200 dark:border-neutral-700"
            >
              <Link
                href={`https://us-east-1.console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/create/review?templateURL=https://guard-cloudformation-templates.s3.amazonaws.com/guard-scan-role.yaml&stackName=GuardSecurityScanRole&param_ExternalId=${externalId}`}
                target="_blank"
                className="flex items-center gap-2"
              >
                <span>Launch CloudFormation Stack</span>
                <svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <path d="M3.64645 11.3536C3.45118 11.1583 3.45118 10.8417 3.64645 10.6464L10.2929 4L6 4C5.72386 4 5.5 3.77614 5.5 3.5C5.5 3.22386 5.72386 3 6 3L11.5 3C11.7761 3 12 3.22386 12 3.5L12 9C12 9.27614 11.7761 9.5 11.5 9.5C11.2239 9.5 11 9.27614 11 9L11 4.70711L4.35355 11.3536C4.15829 11.5488 3.84171 11.5488 3.64645 11.3536Z" fill="currentColor" fillRule="evenodd" clipRule="evenodd"></path>
                </svg>
              </Link>
            </Button>
          </div>
        </div>

        {/* Step 2 */}
        <div className="relative pl-8">
          <div className="absolute left-0 top-1.5 w-4 h-4 rounded-full bg-gradient-to-br from-neutral-800 to-neutral-900 dark:from-neutral-200 dark:to-white flex items-center justify-center">
            <span className="text-xs text-white dark:text-neutral-900 font-bold">2</span>
          </div>
          <div className="space-y-2">
            <h3 className="font-[family-name:var(--font-geist-mono)] text-base">Verify Connection</h3>
            <p className="text-sm text-muted-foreground">Enter your 12-digit AWS Account ID below to verify the connection and complete onboarding.</p>
            <div className="flex items-center gap-4 mt-4">
              <Input
                placeholder="12 Digit AWS Account ID"
                value={accountId}
                onChange={e => setAccountId(e.target.value)}
                className="font-[family-name:var(--font-geist-mono)]"
              />
              <Button
                disabled={!isValidAwsAccountId(accountId) || loading}
                onClick={() => verifyAccountId()}
                className="rounded-full bg-gradient-to-r from-neutral-900 via-neutral-800 to-neutral-700 hover:from-neutral-800 hover:via-neutral-700 hover:to-neutral-600 text-white border-0 dark:from-white dark:via-neutral-200 dark:to-neutral-400 dark:hover:from-neutral-200 dark:hover:via-neutral-300 dark:hover:to-neutral-500 dark:text-neutral-900"
              >
                {loading ? 'Verifying...' : 'Verify Connection'}
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Status Messages */}
      {data?.verifyAccountId === true && (
        <div className="mt-6 p-4 rounded-lg bg-gradient-to-r from-green-100 to-green-50 dark:from-green-900 dark:to-green-800 text-green-800 dark:text-green-100 border border-green-200 dark:border-green-700">
          Connection verified successfully! Redirecting...
        </div>
      )}
      {error && (
        <div className="mt-6 p-4 rounded-lg bg-gradient-to-r from-red-100 to-red-50 dark:from-red-900 dark:to-red-800 text-red-800 dark:text-red-100 border border-red-200 dark:border-red-700">
          Could not verify account. Please make sure that the CloudFormation stack is properly deployed and try again.
        </div>
      )}
    </div>
  );

  return (
    <div className="w-full">
      {process.env.NEXT_PUBLIC_SELF_HOSTING ? renderSelfHostingInstructions() : renderCloudFormationInstructions()}
    </div>
  );
};

export default Installation;
