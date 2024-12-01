'use client';

import { useQuery } from "@apollo/client";
import { GET_SCANS } from "./gql";
import { GetScansQuery } from "@/gql/graphql";
import { useEffect, useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTable } from "./data_table";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

export type SCAN_ITEMS = GetScansQuery["teams"][0]["projects"][0]["scans"][0]["scanItems"][0];

const ScanPage = ({ params }: { params: { teamSlug: string; projectSlug: string; scanId: string } }) => {
  const { teamSlug, projectSlug, scanId } = params;
  const pathname = usePathname();
  const parentPath = pathname.split('/scan/')[0];

  const [scanCompleted, setScanComplete] = useState(false);

  const { data, loading } = useQuery<GetScansQuery>(GET_SCANS, { variables: { teamSlug, projectSlug, scanId }, pollInterval: scanCompleted ? 0 : 5000 });

  const scanData = data?.teams[0].projects[0].scans[0];
  const scans = scanData?.scanItems;
  const scanCompl = scanData?.scanCompleted;

  useEffect(() => {
    if (scanCompl) {
      setScanComplete(true);
    }
  }, [data])

  const LoadingSkeleton = () => (
    <div className="w-full">
      <div className="flex items-center py-4">
        <Skeleton className="h-10 w-[300px]" /> {/* Skeleton for the input field */}
      </div>
      <div className="rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30">
        <Table>
          <TableHeader>
            <TableRow>
              {['Service', 'Region', 'Summary', 'Findings'].map((header) => (
                <TableHead key={header}>
                  <Skeleton className="h-8 w-full" />
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {[...Array(10)].map((_, index) => (
              <TableRow key={index}>
                {[...Array(4)].map((_, cellIndex) => (
                  <TableCell key={cellIndex}>
                    <Skeleton className="h-6 w-full" />
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-end space-x-2 py-4">
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-20" />
      </div>
    </div>
  );

  return (
    <div className="space-y-6 flex w-full h-full items-center justify-center flex-col py-5">
      {!loading ?
        <div className="flex items-center gap-4 w-full max-w-screen-xl h-full flex-col">

          <div className="w-full">
            <Button
              variant="ghost"
              size="sm"
              className="gap-2 pl-0 font-normal"
              asChild
            >
              <Link href={parentPath}>
                <ArrowLeft className="h-4 w-4" />
                Back to All Scans
              </Link>
            </Button>
          </div>

          <DataTable data={scans || []} scanCompleted={scanCompleted} />
        </div>
        :
        <div className="flex w-full h-full max-w-screen-xl flex-col">
          <LoadingSkeleton />
        </div>
      }

    </div>
  );
};

export default ScanPage;
