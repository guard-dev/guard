import { MetadataRoute } from 'next';

const baseUrl = 'https://www.guard.dev';

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  return [
    {
      url: baseUrl,
      lastModified: new Date().toISOString(),
      changeFrequency: 'yearly',
      priority: 1,
    },
  ];
}

