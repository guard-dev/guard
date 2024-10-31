/* eslint-disable */
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  DateTime: { input: any; output: any; }
  Int64: { input: any; output: any; }
  Upload: { input: any; output: any; }
};

export type AccountConnection = {
  __typename?: 'AccountConnection';
  accountId: Scalars['String']['output'];
  externalId: Scalars['String']['output'];
};

export type CheckoutSessionResponse = {
  __typename?: 'CheckoutSessionResponse';
  sessionId: Scalars['String']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  createCheckoutSession: CheckoutSessionResponse;
  createPortalSession: PortalSessionResponse;
  createProject: Project;
  startScan: Scalars['String']['output'];
  verifyAccountId: Scalars['Boolean']['output'];
};


export type MutationCreateCheckoutSessionArgs = {
  lookUpKey: Scalars['String']['input'];
  teamSlug: Scalars['String']['input'];
};


export type MutationCreatePortalSessionArgs = {
  teamSlug: Scalars['String']['input'];
};


export type MutationCreateProjectArgs = {
  input: NewProject;
  teamSlug: Scalars['String']['input'];
};


export type MutationStartScanArgs = {
  projectSlug: Scalars['String']['input'];
  regions: Array<Scalars['String']['input']>;
  services: Array<Scalars['String']['input']>;
  teamSlug: Scalars['String']['input'];
};


export type MutationVerifyAccountIdArgs = {
  accountId: Scalars['String']['input'];
};

export type NewProject = {
  projectName: Scalars['String']['input'];
};

export type PortalSessionResponse = {
  __typename?: 'PortalSessionResponse';
  sessionUrl: Scalars['String']['output'];
};

export type Project = {
  __typename?: 'Project';
  accountConnections: Array<AccountConnection>;
  projectName: Scalars['String']['output'];
  projectSlug: Scalars['String']['output'];
  scans: Array<Scan>;
};


export type ProjectScansArgs = {
  scanId?: InputMaybe<Scalars['String']['input']>;
};

export type Query = {
  __typename?: 'Query';
  getExternalId: Scalars['String']['output'];
  teams: Array<Team>;
};


export type QueryTeamsArgs = {
  teamSlug?: InputMaybe<Scalars['String']['input']>;
};

export type Scan = {
  __typename?: 'Scan';
  created: Scalars['Int64']['output'];
  regionCount: Scalars['Int']['output'];
  resourceCost: Scalars['Int']['output'];
  scanCompleted: Scalars['Boolean']['output'];
  scanId: Scalars['String']['output'];
  scanItems: Array<ScanItem>;
  serviceCount: Scalars['Int']['output'];
};

export type ScanItem = {
  __typename?: 'ScanItem';
  findings: Array<Scalars['String']['output']>;
  region: Scalars['String']['output'];
  remedy: Scalars['String']['output'];
  resourceCost: Scalars['Int']['output'];
  scanItemEntries: Array<ScanItemEntry>;
  service: Scalars['String']['output'];
  summary: Scalars['String']['output'];
};

export type ScanItemEntry = {
  __typename?: 'ScanItemEntry';
  commands: Array<Scalars['String']['output']>;
  findings: Array<Scalars['String']['output']>;
  remedy: Scalars['String']['output'];
  resourceCost: Scalars['Int']['output'];
  summary: Scalars['String']['output'];
  title: Scalars['String']['output'];
};

export type SubscriptionData = {
  __typename?: 'SubscriptionData';
  costInUsd: Scalars['Int64']['output'];
  currentPeriodEnd: Scalars['DateTime']['output'];
  currentPeriodStart: Scalars['DateTime']['output'];
  interval: Scalars['String']['output'];
  lastFourCardDigits: Scalars['String']['output'];
  planName: Scalars['String']['output'];
  resourcesIncluded: Scalars['Int']['output'];
  resourcesUsed: Scalars['Int']['output'];
  status: Scalars['String']['output'];
};

export type SubscriptionPlan = {
  __typename?: 'SubscriptionPlan';
  id: Scalars['Int64']['output'];
  stripeSubscriptionId?: Maybe<Scalars['String']['output']>;
  subscriptionData?: Maybe<SubscriptionData>;
  teamId: Scalars['Int64']['output'];
};

export type Team = {
  __typename?: 'Team';
  members: Array<TeamMembership>;
  projects: Array<Project>;
  subscriptionPlans: Array<SubscriptionPlan>;
  teamName: Scalars['String']['output'];
  teamSlug: Scalars['String']['output'];
};


export type TeamProjectsArgs = {
  projectSlug?: InputMaybe<Scalars['String']['input']>;
};


export type TeamSubscriptionPlansArgs = {
  subscriptionId?: InputMaybe<Scalars['Int64']['input']>;
};

export type TeamMembership = {
  __typename?: 'TeamMembership';
  membershipType: Scalars['String']['output'];
  teamId: Scalars['Int64']['output'];
  teamName: Scalars['String']['output'];
  teamSlug: Scalars['String']['output'];
  user: Userinfo;
};

export type Userinfo = {
  __typename?: 'Userinfo';
  email: Scalars['String']['output'];
  fullName: Scalars['String']['output'];
  userId: Scalars['Int64']['output'];
};

export type CreateStripeCheckoutMutationVariables = Exact<{
  teamSlug: Scalars['String']['input'];
  lookUpKey: Scalars['String']['input'];
}>;


export type CreateStripeCheckoutMutation = { __typename?: 'Mutation', createCheckoutSession: { __typename?: 'CheckoutSessionResponse', sessionId: string } };

export type GetProjectInfoQueryVariables = Exact<{
  teamSlug: Scalars['String']['input'];
  projectSlug: Scalars['String']['input'];
}>;


export type GetProjectInfoQuery = { __typename?: 'Query', teams: Array<{ __typename?: 'Team', subscriptionPlans: Array<{ __typename?: 'SubscriptionPlan', id: any, stripeSubscriptionId?: string | null, subscriptionData?: { __typename?: 'SubscriptionData', currentPeriodStart: any, currentPeriodEnd: any, status: string, interval: string, planName: string, costInUsd: any, lastFourCardDigits: string, resourcesIncluded: number, resourcesUsed: number } | null }>, projects: Array<{ __typename?: 'Project', projectSlug: string, projectName: string, accountConnections: Array<{ __typename?: 'AccountConnection', externalId: string, accountId: string }>, scans: Array<{ __typename?: 'Scan', scanId: string, scanCompleted: boolean, created: any, serviceCount: number, regionCount: number, resourceCost: number }> }> }> };

export type StartScanMutationVariables = Exact<{
  teamSlug: Scalars['String']['input'];
  projectSlug: Scalars['String']['input'];
  regions: Array<Scalars['String']['input']> | Scalars['String']['input'];
  services: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type StartScanMutation = { __typename?: 'Mutation', startScan: string };

export type GetScansQueryVariables = Exact<{
  teamSlug: Scalars['String']['input'];
  projectSlug: Scalars['String']['input'];
  scanId: Scalars['String']['input'];
}>;


export type GetScansQuery = { __typename?: 'Query', teams: Array<{ __typename?: 'Team', projects: Array<{ __typename?: 'Project', scans: Array<{ __typename?: 'Scan', scanCompleted: boolean, serviceCount: number, regionCount: number, resourceCost: number, scanItems: Array<{ __typename?: 'ScanItem', service: string, region: string, findings: Array<string>, summary: string, remedy: string, resourceCost: number, scanItemEntries: Array<{ __typename?: 'ScanItemEntry', findings: Array<string>, title: string, summary: string, remedy: string, commands: Array<string>, resourceCost: number }> }> }> }> }> };

export type CreateStripePortalSessionMutationVariables = Exact<{
  teamSlug: Scalars['String']['input'];
}>;


export type CreateStripePortalSessionMutation = { __typename?: 'Mutation', createPortalSession: { __typename?: 'PortalSessionResponse', sessionUrl: string } };

export type GetExternalIdQueryVariables = Exact<{ [key: string]: never; }>;


export type GetExternalIdQuery = { __typename?: 'Query', getExternalId: string };

export type GetProjectsQueryVariables = Exact<{
  teamSlug?: InputMaybe<Scalars['String']['input']>;
}>;


export type GetProjectsQuery = { __typename?: 'Query', teams: Array<{ __typename?: 'Team', projects: Array<{ __typename?: 'Project', projectSlug: string, projectName: string }> }> };

export type GetTeamsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetTeamsQuery = { __typename?: 'Query', teams: Array<{ __typename?: 'Team', teamName: string, teamSlug: string, projects: Array<{ __typename?: 'Project', projectSlug: string }> }> };

export type VerifyAccountIdMutationVariables = Exact<{
  accountId: Scalars['String']['input'];
}>;


export type VerifyAccountIdMutation = { __typename?: 'Mutation', verifyAccountId: boolean };


export const CreateStripeCheckoutDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateStripeCheckout"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"lookUpKey"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createCheckoutSession"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"teamSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}}},{"kind":"Argument","name":{"kind":"Name","value":"lookUpKey"},"value":{"kind":"Variable","name":{"kind":"Name","value":"lookUpKey"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sessionId"}}]}}]}}]} as unknown as DocumentNode<CreateStripeCheckoutMutation, CreateStripeCheckoutMutationVariables>;
export const GetProjectInfoDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetProjectInfo"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"projectSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"teams"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"teamSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"subscriptionPlans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"stripeSubscriptionId"}},{"kind":"Field","name":{"kind":"Name","value":"subscriptionData"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"currentPeriodStart"}},{"kind":"Field","name":{"kind":"Name","value":"currentPeriodEnd"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"interval"}},{"kind":"Field","name":{"kind":"Name","value":"planName"}},{"kind":"Field","name":{"kind":"Name","value":"costInUsd"}},{"kind":"Field","name":{"kind":"Name","value":"lastFourCardDigits"}},{"kind":"Field","name":{"kind":"Name","value":"resourcesIncluded"}},{"kind":"Field","name":{"kind":"Name","value":"resourcesUsed"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"projects"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"projectSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"projectSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"projectSlug"}},{"kind":"Field","name":{"kind":"Name","value":"projectName"}},{"kind":"Field","name":{"kind":"Name","value":"accountConnections"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"externalId"}},{"kind":"Field","name":{"kind":"Name","value":"accountId"}}]}},{"kind":"Field","name":{"kind":"Name","value":"scans"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"scanId"}},{"kind":"Field","name":{"kind":"Name","value":"scanCompleted"}},{"kind":"Field","name":{"kind":"Name","value":"created"}},{"kind":"Field","name":{"kind":"Name","value":"serviceCount"}},{"kind":"Field","name":{"kind":"Name","value":"regionCount"}},{"kind":"Field","name":{"kind":"Name","value":"resourceCost"}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetProjectInfoQuery, GetProjectInfoQueryVariables>;
export const StartScanDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"StartScan"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"projectSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"regions"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"services"}},"type":{"kind":"NonNullType","type":{"kind":"ListType","type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"startScan"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"teamSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}}},{"kind":"Argument","name":{"kind":"Name","value":"projectSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"projectSlug"}}},{"kind":"Argument","name":{"kind":"Name","value":"services"},"value":{"kind":"Variable","name":{"kind":"Name","value":"services"}}},{"kind":"Argument","name":{"kind":"Name","value":"regions"},"value":{"kind":"Variable","name":{"kind":"Name","value":"regions"}}}]}]}}]} as unknown as DocumentNode<StartScanMutation, StartScanMutationVariables>;
export const GetScansDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetScans"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"projectSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}},{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"scanId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"teams"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"teamSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"projects"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"projectSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"projectSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"scans"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"scanId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"scanId"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"scanCompleted"}},{"kind":"Field","name":{"kind":"Name","value":"serviceCount"}},{"kind":"Field","name":{"kind":"Name","value":"regionCount"}},{"kind":"Field","name":{"kind":"Name","value":"resourceCost"}},{"kind":"Field","name":{"kind":"Name","value":"scanItems"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"service"}},{"kind":"Field","name":{"kind":"Name","value":"region"}},{"kind":"Field","name":{"kind":"Name","value":"findings"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"remedy"}},{"kind":"Field","name":{"kind":"Name","value":"resourceCost"}},{"kind":"Field","name":{"kind":"Name","value":"scanItemEntries"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"findings"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"summary"}},{"kind":"Field","name":{"kind":"Name","value":"remedy"}},{"kind":"Field","name":{"kind":"Name","value":"commands"}},{"kind":"Field","name":{"kind":"Name","value":"resourceCost"}}]}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<GetScansQuery, GetScansQueryVariables>;
export const CreateStripePortalSessionDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"CreateStripePortalSession"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"createPortalSession"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"teamSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"sessionUrl"}}]}}]}}]} as unknown as DocumentNode<CreateStripePortalSessionMutation, CreateStripePortalSessionMutationVariables>;
export const GetExternalIdDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetExternalId"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"getExternalId"}}]}}]} as unknown as DocumentNode<GetExternalIdQuery, GetExternalIdQueryVariables>;
export const GetProjectsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetProjects"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}},"type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"teams"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"teamSlug"},"value":{"kind":"Variable","name":{"kind":"Name","value":"teamSlug"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"projects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"projectSlug"}},{"kind":"Field","name":{"kind":"Name","value":"projectName"}}]}}]}}]}}]} as unknown as DocumentNode<GetProjectsQuery, GetProjectsQueryVariables>;
export const GetTeamsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"GetTeams"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"teams"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"teamName"}},{"kind":"Field","name":{"kind":"Name","value":"teamSlug"}},{"kind":"Field","name":{"kind":"Name","value":"projects"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"projectSlug"}}]}}]}}]}}]} as unknown as DocumentNode<GetTeamsQuery, GetTeamsQueryVariables>;
export const VerifyAccountIdDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"VerifyAccountId"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"accountId"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"String"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"verifyAccountId"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"accountId"},"value":{"kind":"Variable","name":{"kind":"Name","value":"accountId"}}}]}]}}]} as unknown as DocumentNode<VerifyAccountIdMutation, VerifyAccountIdMutationVariables>;