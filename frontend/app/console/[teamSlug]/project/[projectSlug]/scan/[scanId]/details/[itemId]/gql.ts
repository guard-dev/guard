import { gql } from "@apollo/client";

export const GET_SCAN_ITEM = gql`
  query GetScanItem($teamSlug: String!, $projectSlug: String!, $scanId: String!, $scanItemId: Int64!) {
    teams(teamSlug: $teamSlug) {
      projects(projectSlug: $projectSlug) {
        scans(scanId: $scanId) {
          scanItems(scanItemId: $scanItemId) {
            scanItemId
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
