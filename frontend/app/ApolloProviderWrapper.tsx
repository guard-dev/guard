'use client';

import { GetApolloClient } from '@/apollo-client';
import { ApolloProvider } from '@apollo/client';
import { useAuth } from '@clerk/nextjs';
import { PropsWithChildren } from 'react';

const ClerkApolloProvider = ({ children }: PropsWithChildren) => {
  const { getToken } = useAuth();
  const client = GetApolloClient(false, getToken);
  return <ApolloProvider client={client}>{children}</ApolloProvider>;
};

const SelfHostedApolloProvider = ({ children }: PropsWithChildren) => {
  const client = GetApolloClient(false, () => Promise.resolve('self-hosted'));
  return <ApolloProvider client={client}>{children}</ApolloProvider>;
};

export const ApolloProviderWrapper = ({ children }: PropsWithChildren) => {
  const selfHosting = process.env.NEXT_PUBLIC_SELF_HOSTING === 'true';

  if (selfHosting) {
    return <SelfHostedApolloProvider>{children}</SelfHostedApolloProvider>;
  }

  return <ClerkApolloProvider>{children}</ClerkApolloProvider>;
};
