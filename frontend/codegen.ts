import { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  schema: '../backend/graph/schema.graphqls',
  documents: ['app/**/*.{tsx,ts}'],
  generates: {
    './gql/': {
      preset: 'client',
    },
  },
};
export default config;
