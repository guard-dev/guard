'use client';

import { useQuery } from "@apollo/client";
import Installation from "./Installation";
import { GetExternalIdQuery, GetTeamsQuery } from "@/gql/graphql";
import { GET_EXTERNAL_ID, GET_TEAMS } from "./gql";
import { useEffect } from "react";
import { useRouter } from "next/navigation";

const ConsolePage = () => {
  const { data: team, refetch } = useQuery<GetTeamsQuery>(GET_TEAMS, { variables: { teamSlug: undefined } });
  const { data } = useQuery<GetExternalIdQuery>(GET_EXTERNAL_ID);
  const externalId = data?.getExternalId;

  const router = useRouter();
  useEffect(() => {
    if (team?.teams.length) {
      router.push(`/console/${team?.teams[0].teamSlug}/project/${team?.teams[0].projects[0].projectSlug}`)
    }
  }, [team]);

  return (
    <div className="flex w-full h-full bg-background">
      <div className="flex w-full h-full items-center justify-center p-8">
        {team?.teams.length === 0 && (
          <div className="w-full max-w-3xl">
            {/* Header */}
            <div className="space-y-4 mb-12">
              <h1 className="text-4xl font-[family-name:var(--font-geist-mono)] bg-gradient-to-r from-neutral-900 to-neutral-600 dark:from-white dark:to-neutral-400 bg-clip-text text-transparent">
                {process.env.NEXT_PUBLIC_SELF_HOSTING ? "Set Up Your IAM User" : "Connect your AWS Account"}
              </h1>
              <div className="space-y-4 text-lg text-muted-foreground">
                <p>
                  {process.env.NEXT_PUBLIC_SELF_HOSTING
                    ? "To allow Guard to scan your AWS account, you need to create an IAM User with read-only permissions. This IAM User will allow Guard to securely access your AWS resources for security scans without modifying them."
                    : "To allow Guard to scan your AWS account, you'll need to add a read-only permission using AWS CloudFormation. This will allow Guard to securely access your AWS resources for security scans without modifying them."}
                </p>
                <div className="flex items-center gap-2 p-4 rounded-lg bg-gradient-to-br from-yellow-50 via-yellow-100/80 to-yellow-50 dark:from-yellow-950 dark:via-yellow-900/50 dark:to-yellow-950/30 border border-yellow-200 dark:border-yellow-800">
                  <svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg" className="flex-shrink-0">
                    <path d="M8.4449 0.608765C8.0183 -0.107015 6.9817 -0.107015 6.5551 0.608766L0.161018 11.3368C-0.275725 12.07 0.252503 13 1.10592 13H13.8941C14.7475 13 15.2757 12.07 14.839 11.3368L8.4449 0.608765ZM7.4141 1.12073C7.4141 1.12073 7.4141 1.12073 7.4141 1.12073L1.02002 11.8488C0.972117 11.9337 1.03191 12.0392 1.10592 12.0392H13.8941C13.9681 12.0392 14.0279 11.9337 13.98 11.8488L7.5859 1.12073C7.5859 1.12073 7.5859 1.12073 7.5859 1.12073C7.54475 1.05154 7.45525 1.05154 7.4141 1.12073ZM7 4.5C7.27614 4.5 7.5 4.72386 7.5 5V8C7.5 8.27614 7.27614 8.5 7 8.5C6.72386 8.5 6.5 8.27614 6.5 8V5C6.5 4.72386 6.72386 4.5 7 4.5ZM7 11C7.27614 11 7.5 10.7761 7.5 10.5C7.5 10.2239 7.27614 10 7 10C6.72386 10 6.5 10.2239 6.5 10.5C6.5 10.7761 6.72386 11 7 11Z" fill="currentColor" fillRule="evenodd" clipRule="evenodd"></path>
                  </svg>
                  <p className="text-sm font-medium">
                    Scanning your account will not incur any charges to your AWS account.
                  </p>
                </div>
              </div>
            </div>

            {/* Installation Steps */}
            <div className="rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 p-8">
              <Installation externalId={externalId || ""} refetch={refetch} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default ConsolePage;
