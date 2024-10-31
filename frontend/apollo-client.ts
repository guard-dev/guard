import { ApolloClient, InMemoryCache, from } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { createUploadLink } from 'apollo-upload-client';

export const GetApolloClient = (ssrMode: boolean, getToken: any) => {
  const apiServer =
    process.env.NEXT_SERVER_API_BASE_URL ||
    process.env.NEXT_PUBLIC_API_BASE_URL;

  const vercel_environent =
    process.env.VERCEL_ENV || process.env.NEXT_PUBLIC_VERCEL_ENV;

  const vercel_env = vercel_environent ? vercel_environent : 'development';

  const httpLink: any = createUploadLink({ uri: apiServer });

  const authMiddleware = setContext(async (_, { headers }) => {
    const token = await getToken({ template: 'Guard_GQL_Server' });
    return {
      headers: {
        ...headers,
        authorization: token ? `Bearer ${token}` : '',
        vercel_env,
      },
    };
  });
  return new ApolloClient({
    ssrMode,
    link: from([authMiddleware, httpLink]),
    cache: new InMemoryCache(),
  });
};
