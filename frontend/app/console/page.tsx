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
    <div className="flex w-full h-full">
      <div className="flex w-full h-full items-center justify-center flex-col pt-[50px] p-5">
        {team?.teams.length === 0 &&
          <div className="w-full max-w-[700px] gap-8 flex flex-col">
            <h1 className="text-4xl w-full">
              {process.env.NEXT_PUBLIC_SELF_HOSTING ? "Set Up Your IAM User" : "Connect your AWS Account"}
            </h1>
            <div className="w-full flex flex-col gap-4 text-lg">
              <p className="w-full">
                {process.env.NEXT_PUBLIC_SELF_HOSTING
                  ? "To allow Guard to scan your AWS account, you need to create an IAM User with read-only permissions. This IAM User will allow Guard to securely access your AWS resources for security scans without modifying them."
                  : "To allow Guard to scan your AWS account, you’ll need to add a read-only permission using AWS CloudFormation. This will allow Guard to securely access your AWS resources for security scans without modifying them."}
              </p>
              <p className="font-bold">
                Note: Scanning your account will not incur any charges to your AWS account.
              </p>
            </div>
            <Installation externalId={externalId || ""} refetch={refetch} />
          </div>
        }
      </div>
    </div>
  );
}

export default ConsolePage;

