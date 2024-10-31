/* eslint-disable */
import * as types from './graphql';
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';

/**
 * Map of all GraphQL operations in the project.
 *
 * This map has several performance disadvantages:
 * 1. It is not tree-shakeable, so it will include all operations in the project.
 * 2. It is not minifiable, so the string of a GraphQL query will be multiple times inside the bundle.
 * 3. It does not support dead code elimination, so it will add unused operations.
 *
 * Therefore it is highly recommended to use the babel or swc plugin for production.
 */
const documents = {
    "\n  mutation CreateStripeCheckout($teamSlug: String!, $lookUpKey: String!) {\n    createCheckoutSession(teamSlug: $teamSlug, lookUpKey: $lookUpKey) {\n      sessionId\n    }\n  }\n": types.CreateStripeCheckoutDocument,
    "\n  query GetProjectInfo($teamSlug: String!, $projectSlug: String!) {\n    teams(teamSlug: $teamSlug) {\n      subscriptionPlans {\n        id\n        stripeSubscriptionId\n        subscriptionData {\n          currentPeriodStart\n          currentPeriodEnd\n          status\n          interval\n          planName\n          costInUsd\n          lastFourCardDigits\n          resourcesIncluded\n          resourcesUsed\n        }\n      }\n      projects(projectSlug: $projectSlug) {\n        projectSlug\n        projectName\n        accountConnections {\n          externalId\n          accountId\n        }\n        scans {\n          scanId\n          scanCompleted\n          created\n          serviceCount\n          regionCount\n          resourceCost\n        }\n      }\n    }\n  }\n": types.GetProjectInfoDocument,
    "\n  mutation StartScan($teamSlug: String!, $projectSlug: String!, $regions: [String!]!, $services: [String!]!) {\n    startScan(teamSlug: $teamSlug, projectSlug: $projectSlug, services: $services, regions: $regions)\n  }\n": types.StartScanDocument,
    "\n  query GetScans($teamSlug: String!, $projectSlug: String!, $scanId: String!) {\n    teams(teamSlug: $teamSlug) {\n      projects(projectSlug: $projectSlug) {\n        scans(scanId: $scanId) {\n          scanCompleted\n          serviceCount\n          regionCount\n          resourceCost\n          scanItems {\n            service\n            region\n            findings\n            summary\n            remedy\n            resourceCost\n            scanItemEntries {\n              findings\n              title\n              summary\n              remedy\n              commands\n              resourceCost\n            }\n          }\n        }\n      }\n    }\n  }\n": types.GetScansDocument,
    "\n  mutation CreateStripePortalSession($teamSlug: String!) {\n    createPortalSession(teamSlug: $teamSlug) {\n      sessionUrl\n    }\n  }\n": types.CreateStripePortalSessionDocument,
    "\n  query GetExternalId {\n    getExternalId\n  }\n": types.GetExternalIdDocument,
    "\n  query GetProjects($teamSlug: String) {\n    teams(teamSlug: $teamSlug) {\n      projects {\n        projectSlug\n        projectName\n      }\n    }\n  }\n": types.GetProjectsDocument,
    "\n  query GetTeams {\n    teams {\n      teamName\n      teamSlug\n      projects {\n        projectSlug\n      }\n    }\n  }\n": types.GetTeamsDocument,
    "\n  mutation VerifyAccountId($accountId: String!) {\n    verifyAccountId(accountId:$accountId)\n  }\n": types.VerifyAccountIdDocument,
};

/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 *
 *
 * @example
 * ```ts
 * const query = graphql(`query GetUser($id: ID!) { user(id: $id) { name } }`);
 * ```
 *
 * The query argument is unknown!
 * Please regenerate the types.
 */
export function graphql(source: string): unknown;

/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation CreateStripeCheckout($teamSlug: String!, $lookUpKey: String!) {\n    createCheckoutSession(teamSlug: $teamSlug, lookUpKey: $lookUpKey) {\n      sessionId\n    }\n  }\n"): (typeof documents)["\n  mutation CreateStripeCheckout($teamSlug: String!, $lookUpKey: String!) {\n    createCheckoutSession(teamSlug: $teamSlug, lookUpKey: $lookUpKey) {\n      sessionId\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query GetProjectInfo($teamSlug: String!, $projectSlug: String!) {\n    teams(teamSlug: $teamSlug) {\n      subscriptionPlans {\n        id\n        stripeSubscriptionId\n        subscriptionData {\n          currentPeriodStart\n          currentPeriodEnd\n          status\n          interval\n          planName\n          costInUsd\n          lastFourCardDigits\n          resourcesIncluded\n          resourcesUsed\n        }\n      }\n      projects(projectSlug: $projectSlug) {\n        projectSlug\n        projectName\n        accountConnections {\n          externalId\n          accountId\n        }\n        scans {\n          scanId\n          scanCompleted\n          created\n          serviceCount\n          regionCount\n          resourceCost\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query GetProjectInfo($teamSlug: String!, $projectSlug: String!) {\n    teams(teamSlug: $teamSlug) {\n      subscriptionPlans {\n        id\n        stripeSubscriptionId\n        subscriptionData {\n          currentPeriodStart\n          currentPeriodEnd\n          status\n          interval\n          planName\n          costInUsd\n          lastFourCardDigits\n          resourcesIncluded\n          resourcesUsed\n        }\n      }\n      projects(projectSlug: $projectSlug) {\n        projectSlug\n        projectName\n        accountConnections {\n          externalId\n          accountId\n        }\n        scans {\n          scanId\n          scanCompleted\n          created\n          serviceCount\n          regionCount\n          resourceCost\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation StartScan($teamSlug: String!, $projectSlug: String!, $regions: [String!]!, $services: [String!]!) {\n    startScan(teamSlug: $teamSlug, projectSlug: $projectSlug, services: $services, regions: $regions)\n  }\n"): (typeof documents)["\n  mutation StartScan($teamSlug: String!, $projectSlug: String!, $regions: [String!]!, $services: [String!]!) {\n    startScan(teamSlug: $teamSlug, projectSlug: $projectSlug, services: $services, regions: $regions)\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query GetScans($teamSlug: String!, $projectSlug: String!, $scanId: String!) {\n    teams(teamSlug: $teamSlug) {\n      projects(projectSlug: $projectSlug) {\n        scans(scanId: $scanId) {\n          scanCompleted\n          serviceCount\n          regionCount\n          resourceCost\n          scanItems {\n            service\n            region\n            findings\n            summary\n            remedy\n            resourceCost\n            scanItemEntries {\n              findings\n              title\n              summary\n              remedy\n              commands\n              resourceCost\n            }\n          }\n        }\n      }\n    }\n  }\n"): (typeof documents)["\n  query GetScans($teamSlug: String!, $projectSlug: String!, $scanId: String!) {\n    teams(teamSlug: $teamSlug) {\n      projects(projectSlug: $projectSlug) {\n        scans(scanId: $scanId) {\n          scanCompleted\n          serviceCount\n          regionCount\n          resourceCost\n          scanItems {\n            service\n            region\n            findings\n            summary\n            remedy\n            resourceCost\n            scanItemEntries {\n              findings\n              title\n              summary\n              remedy\n              commands\n              resourceCost\n            }\n          }\n        }\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation CreateStripePortalSession($teamSlug: String!) {\n    createPortalSession(teamSlug: $teamSlug) {\n      sessionUrl\n    }\n  }\n"): (typeof documents)["\n  mutation CreateStripePortalSession($teamSlug: String!) {\n    createPortalSession(teamSlug: $teamSlug) {\n      sessionUrl\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query GetExternalId {\n    getExternalId\n  }\n"): (typeof documents)["\n  query GetExternalId {\n    getExternalId\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query GetProjects($teamSlug: String) {\n    teams(teamSlug: $teamSlug) {\n      projects {\n        projectSlug\n        projectName\n      }\n    }\n  }\n"): (typeof documents)["\n  query GetProjects($teamSlug: String) {\n    teams(teamSlug: $teamSlug) {\n      projects {\n        projectSlug\n        projectName\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  query GetTeams {\n    teams {\n      teamName\n      teamSlug\n      projects {\n        projectSlug\n      }\n    }\n  }\n"): (typeof documents)["\n  query GetTeams {\n    teams {\n      teamName\n      teamSlug\n      projects {\n        projectSlug\n      }\n    }\n  }\n"];
/**
 * The graphql function is used to parse GraphQL queries into a document that can be used by GraphQL clients.
 */
export function graphql(source: "\n  mutation VerifyAccountId($accountId: String!) {\n    verifyAccountId(accountId:$accountId)\n  }\n"): (typeof documents)["\n  mutation VerifyAccountId($accountId: String!) {\n    verifyAccountId(accountId:$accountId)\n  }\n"];

export function graphql(source: string) {
  return (documents as any)[source] ?? {};
}

export type DocumentType<TDocumentNode extends DocumentNode<any, any>> = TDocumentNode extends DocumentNode<  infer TType,  any>  ? TType  : never;