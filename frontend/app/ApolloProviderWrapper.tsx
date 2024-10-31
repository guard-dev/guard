'use client';

import { GetApolloClient } from '@/apollo-client';
import { ApolloProvider } from '@apollo/client';
import { useAuth } from '@clerk/nextjs';
import { PropsWithChildren, useMemo } from 'react';

export const ApolloProviderWrapper = ({ children }: PropsWithChildren) => {
  const { getToken } = useAuth();
  const client = useMemo(() => {
    return GetApolloClient(false, getToken);
  }, [getToken]);
  return <ApolloProvider client={client}>{children}</ApolloProvider>;
};
