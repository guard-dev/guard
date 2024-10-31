import { gql } from "@apollo/client";

export const GET_SCANS = gql`
  query GetScans($teamSlug: String!, $projectSlug: String!, $scanId: String!) {
    teams(teamSlug: $teamSlug) {
      projects(projectSlug: $projectSlug) {
        scans(scanId: $scanId) {
          scanCompleted
          serviceCount
          regionCount
          resourceCost
          scanItems {
            service
            region
            findings
            summary
            remedy
            resourceCost
            scanItemEntries {
              findings
              title
              summary
              remedy
              commands
              resourceCost
            }
          }
        }
      }
    }
  }
`;

