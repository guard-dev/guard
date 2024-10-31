
import { gql } from '@apollo/client';

export const GET_EXTERNAL_ID = gql`
  query GetExternalId {
    getExternalId
  }
`

export const GET_PROJECTS = gql`
  query GetProjects($teamSlug: String) {
    teams(teamSlug: $teamSlug) {
      projects {
        projectSlug
        projectName
      }
    }
  }
`;

export const GET_TEAMS = gql`
  query GetTeams {
    teams {
      teamName
      teamSlug
      projects {
        projectSlug
      }
    }
  }
`;

export const VERIFY_ACCOUNT_ID = gql`
  mutation VerifyAccountId($accountId: String!) {
    verifyAccountId(accountId:$accountId)
  }
`
