'use client';

import { useQuery } from "@apollo/client";
import { GET_SCAN_ITEM } from "./gql";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { ArrowLeft, AlertCircle, Terminal, Shield, FileText } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

interface ScanItemEntry {
  findings: string[];
  title: string;
  summary: string;
  remedy: string;
  commands: string[];
  resourceCost: number;
}

interface ScanItem {
  scanItemId: number;
  service: string;
  region: string;
  findings: string[];
  summary: string;
  remedy: string;
  resourceCost: number;
  scanItemEntries: ScanItemEntry[];
}

export default function DetailsPage({
  params
}: {
  params: {
    teamSlug: string;
    projectSlug: string;
    scanId: string;
    itemId: string;
  }
}) {
  const { teamSlug, projectSlug, scanId, itemId } = params;
  const pathname = usePathname();
  const parentPath = pathname.split('/details/')[0];

  const { data, loading, error } = useQuery(GET_SCAN_ITEM, {
    variables: {
      teamSlug,
      projectSlug,
      scanId,
      scanItemId: parseInt(itemId)
    },
  });

  if (loading) {
    return <LoadingSkeleton />;
  }

  if (error) {
    return (
      <div className="flex flex-col gap-4 p-6">
        <Link href={parentPath}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Scan Results
          </Button>
        </Link>
        <div className="text-red-500">Error loading scan item details</div>
      </div>
    );
  }

  const scanItem: ScanItem = data?.teams[0]?.projects[0]?.scans[0]?.scanItems[0];

  if (!scanItem) {
    return (
      <div className="flex flex-col gap-4 p-6">
        <Link href={parentPath}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Scan Results
          </Button>
        </Link>
        <div className="text-red-500">Scan item not found</div>
      </div>
    );
  }

  return (
    <div className="flex w-full min-h-screen bg-background">
      <div className="flex-1 flex flex-col max-w-screen-2xl mx-auto px-6 py-8">
        <div className="flex items-center gap-4 mb-8">
          <Link href={parentPath}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Scan Results
            </Button>
          </Link>
          <Separator orientation="vertical" className="h-6" />
          <div>
            <Badge variant="outline" className="mr-2">
              {scanItem.service}
            </Badge>
            <Badge variant="secondary">{scanItem.region}</Badge>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main Content - Left Side */}
          <div className="lg:col-span-2 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <AlertCircle className="h-5 w-5 text-orange-500" />
                  Overview
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="prose prose-sm max-w-none">
                  <p className="whitespace-pre-wrap text-muted-foreground">{scanItem.summary}</p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Shield className="h-5 w-5 text-red-500" />
                  Findings
                </CardTitle>
                <CardDescription>
                  Detailed analysis of security findings and recommendations
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-8">
                  {scanItem.scanItemEntries.map((entry, idx) => (
                    <div key={idx} className="space-y-4">
                      <div className="flex items-start gap-4">
                        <div className="h-8 w-8 rounded-full bg-red-100 flex items-center justify-center">
                          <span className="text-sm font-medium text-red-600">{idx + 1}</span>
                        </div>
                        <div className="flex-1">
                          <h3 className="text-lg font-semibold mb-2">{entry.title}</h3>
                          <p className="text-muted-foreground whitespace-pre-wrap mb-4">
                            {entry.summary}
                          </p>

                          <div className="bg-muted/50 rounded-lg p-4 space-y-4">
                            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                              <FileText className="h-4 w-4" />
                              Remediation Steps
                            </div>
                            <p className="text-sm whitespace-pre-wrap">
                              {entry.remedy}
                            </p>
                          </div>

                          {entry.commands.length > 0 && (
                            <div className="mt-4">
                              <div className="flex items-center gap-2 mb-2 text-sm font-medium text-muted-foreground">
                                <Terminal className="h-4 w-4" />
                                AWS CLI Commands
                              </div>
                              <div className="bg-neutral-800 text-neutral-100 rounded-lg p-4 font-mono text-sm">
                                {entry.commands.map((cmd, cmdIdx) => (
                                  <div key={cmdIdx} className="mb-2 last:mb-0">
                                    <code>{cmd}</code>
                                  </div>
                                ))}
                              </div>
                              <p className="text-xs text-muted-foreground mt-2">
                                {`
                                Disclaimer: These commands are for guidance only. Please review and validate them
                                to ensure they align with your environment's specific configurations.
                                `}
                                <Link
                                  href="https://docs.aws.amazon.com/cli/latest/reference/#available-services"
                                  target="_blank"
                                  className="text-blue-700 underline"
                                >
                                  Refer to AWS CLI Documentation
                                </Link>
                              </p>
                            </div>
                          )}
                        </div>
                      </div>
                      {idx < scanItem.scanItemEntries.length - 1 && (
                        <Separator className="my-6" />
                      )}
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Sidebar - Right Side */}
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Resource Summary</CardTitle>
              </CardHeader>
              <CardContent>
                <dl className="space-y-4">
                  <div>
                    <dt className="text-sm font-medium text-muted-foreground">Service</dt>
                    <dd className="text-lg">{scanItem.service}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-muted-foreground">Region</dt>
                    <dd className="text-lg">{scanItem.region}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-muted-foreground">Resource Cost</dt>
                    <dd className="text-lg">{scanItem.resourceCost}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-muted-foreground">Total Findings</dt>
                    <dd className="text-lg">{scanItem.findings.length}</dd>
                  </div>
                </dl>
              </CardContent>
            </Card>

            {scanItem.remedy && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="h-5 w-5 text-green-500" />
                    Quick Fix
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm whitespace-pre-wrap text-muted-foreground">
                    {scanItem.remedy}
                  </p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="flex w-full min-h-screen bg-background">
      <div className="flex-1 flex flex-col max-w-screen-2xl mx-auto px-6 py-8">
        <div className="flex items-center gap-4 mb-8">
          <Skeleton className="h-9 w-36" />
          <Skeleton className="h-6 w-24" />
          <Skeleton className="h-6 w-24" />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <Card>
              <CardHeader>
                <Skeleton className="h-7 w-32" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-24 w-full" />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <Skeleton className="h-7 w-32" />
                <Skeleton className="h-5 w-64" />
              </CardHeader>
              <CardContent>
                {[1, 2, 3].map((i) => (
                  <div key={i} className="space-y-4 mb-6">
                    <div className="flex gap-4">
                      <Skeleton className="h-8 w-8 rounded-full" />
                      <div className="flex-1">
                        <Skeleton className="h-6 w-64 mb-2" />
                        <Skeleton className="h-20 w-full" />
                        <Skeleton className="h-32 w-full mt-4" />
                      </div>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>

          <div className="space-y-6">
            <Card>
              <CardHeader>
                <Skeleton className="h-7 w-40" />
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {[1, 2, 3, 4].map((i) => (
                    <div key={i}>
                      <Skeleton className="h-5 w-32 mb-1" />
                      <Skeleton className="h-7 w-48" />
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
