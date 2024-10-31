import { gql } from "@apollo/client";

export const GET_PROJECT_INFO = gql`
  query GetProjectInfo($teamSlug: String!, $projectSlug: String!) {
    teams(teamSlug: $teamSlug) {
      subscriptionPlans {
        id
        stripeSubscriptionId
        subscriptionData {
          currentPeriodStart
          currentPeriodEnd
          status
          interval
          planName
          costInUsd
          lastFourCardDigits
          resourcesIncluded
          resourcesUsed
        }
      }
      projects(projectSlug: $projectSlug) {
        projectSlug
        projectName
        accountConnections {
          externalId
          accountId
        }
        scans {
          scanId
          scanCompleted
          created
          serviceCount
          regionCount
          resourceCost
        }
      }
    }
  }
`

export const START_SCAN = gql`
  mutation StartScan($teamSlug: String!, $projectSlug: String!, $regions: [String!]!, $services: [String!]!) {
    startScan(teamSlug: $teamSlug, projectSlug: $projectSlug, services: $services, regions: $regions)
  }
`;
