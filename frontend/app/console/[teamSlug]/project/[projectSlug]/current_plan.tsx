import { Badge } from "@/components/ui/badge";
import { gql, useMutation } from "@apollo/client";
import StripeCheckoutForm from "./stripe_checkout_form";
import { CreateStripeCheckoutMutation, GetProjectInfoQuery } from "@/gql/graphql";
import { titleCase } from "@/lib/utils";
import { loadStripe } from '@stripe/stripe-js';

const CREATE_STRIPE_CHECKOUT = gql`
  mutation CreateStripeCheckout($teamSlug: String!, $lookUpKey: String!) {
    createCheckoutSession(teamSlug: $teamSlug, lookUpKey: $lookUpKey) {
      sessionId
    }
  }
`;

type SUBSCRIPTION_PLANS =
  GetProjectInfoQuery['teams'][0]['subscriptionPlans'][0];

export const ShowSubscriptionInfo = ({
  plan,
  teamSlug,
}: {
  plan: SUBSCRIPTION_PLANS;
  teamSlug: string;
}) => {
  const subActive = plan !== undefined && plan?.stripeSubscriptionId !== null;
  const subscriptionData = plan?.subscriptionData;
  const subStatus = subscriptionData?.status;

  const [createCheckoutSession, { }] =
    useMutation<CreateStripeCheckoutMutation>(CREATE_STRIPE_CHECKOUT);

  const handleCheckout = async (lookUpKey: string) => {
    const stripe = await loadStripe(
      process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || '',
    );
    const response = await createCheckoutSession({
      variables: { teamSlug, lookUpKey },
    });
    const sessionId = response.data?.createCheckoutSession?.sessionId || "";
    const res = await stripe?.redirectToCheckout({ sessionId });
    if (res?.error) {
      console.log('[stripe error]', res.error.message);
    }
  };

  const resourcesUsed = subscriptionData?.resourcesUsed;
  const resourcesIncluded = subscriptionData?.resourcesIncluded;

  return (
    <div className="w-full space-y-6">
      {/* Subscription Status */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {subActive ? (
            <>
              <Badge variant="outline" className="bg-gradient-to-r from-green-100 to-green-50 dark:from-green-900 dark:to-green-800 text-green-800 dark:text-green-100 border-green-200 dark:border-green-700">
                {titleCase(subStatus!)}
              </Badge>
              <span className="text-lg font-[family-name:var(--font-geist-mono)]">
                {subscriptionData?.planName}
              </span>
            </>
          ) : (
            <>
              <Badge variant="outline" className="bg-gradient-to-r from-yellow-100 to-yellow-50 dark:from-yellow-900 dark:to-yellow-800 text-yellow-800 dark:text-yellow-100 border-yellow-200 dark:border-yellow-700">
                Inactive
              </Badge>
              <span className="text-lg font-[family-name:var(--font-geist-mono)]">
                No Active Subscription
              </span>
            </>
          )}
        </div>
        {!subActive && (
          <StripeCheckoutForm handleCheckout={handleCheckout} teamSlug={teamSlug} subscriptionActive={subActive} />
        )}
      </div>

      {/* Subscription Details */}
      {subActive && (
        <div className="space-y-6">
          {/* Billing Cycle */}
          <div className="rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 p-4">
            <h3 className="text-sm font-[family-name:var(--font-geist-mono)] text-muted-foreground mb-2">
              Current Billing Cycle
            </h3>
            <p className="font-[family-name:var(--font-geist-mono)]">
              {convertUtcToLocal(subscriptionData?.currentPeriodStart || '')} to{' '}
              {convertUtcToLocal(subscriptionData?.currentPeriodEnd || '')}
            </p>
          </div>

          {/* Resources Usage */}
          <div className="rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 p-4">
            <h3 className="text-sm font-[family-name:var(--font-geist-mono)] text-muted-foreground mb-2">
              Resources Usage
            </h3>
            <div className="space-y-2">
              <div className="w-full h-2 rounded-full bg-neutral-200 dark:bg-neutral-800">
                <div
                  className="h-full rounded-full bg-gradient-to-r from-blue-500 to-blue-600"
                  style={{
                    width: `${Math.min((resourcesUsed! / resourcesIncluded!) * 100, 100)}%`
                  }}
                />
              </div>
              <p className="text-sm font-[family-name:var(--font-geist-mono)]">
                <span className="text-blue-600 dark:text-blue-400">{resourcesUsed}</span>
                <span className="text-muted-foreground"> used of </span>
                <span>{resourcesIncluded}</span>
                <span className="text-muted-foreground"> included</span>
              </p>
            </div>
          </div>

          {/* Subscription Status */}
          {subStatus === 'active' ? (
            <div className="rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 p-4 space-y-3">
              <div className="space-y-1">
                <h3 className="text-sm font-[family-name:var(--font-geist-mono)] text-muted-foreground">
                  Next Renewal
                </h3>
                <p className="font-[family-name:var(--font-geist-mono)]">
                  {convertUtcToLocal(subscriptionData?.currentPeriodEnd || '')}{' '}
                  <span className="text-sm text-muted-foreground">
                    (renews every {subscriptionData?.interval})
                  </span>
                </p>
              </div>

              <div className="space-y-1">
                <h3 className="text-sm font-[family-name:var(--font-geist-mono)] text-muted-foreground">
                  Payment Method
                </h3>
                <p className="font-[family-name:var(--font-geist-mono)]">
                  Card ending in {subscriptionData?.lastFourCardDigits}{' '}
                  <span className="text-sm text-muted-foreground">
                    will be charged ${subscriptionData?.costInUsd}
                  </span>
                </p>
              </div>

              <p className="text-sm text-muted-foreground">
                Click on Manage Billing to update payment information
              </p>
            </div>
          ) : (
            <div className="rounded-lg border bg-gradient-to-br from-red-50 via-red-100/80 to-red-50 dark:from-red-950 dark:via-red-900/50 dark:to-red-950/30 p-4 text-red-800 dark:text-red-200 border-red-200 dark:border-red-800">
              <p className="font-[family-name:var(--font-geist-mono)]">
                There is a problem in renewing your subscription.
              </p>
              <p className="text-sm mt-1">
                Please update your payment information to continue using the service.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

function convertUtcToLocal(utcTimestamp: string): string {
  if (utcTimestamp === '') {
    return '';
  }

  // Create a Date object using the provided UTC timestamp
  const date = new Date(utcTimestamp);

  // Get the month, day, and year
  const monthNames = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ];
  const month = monthNames[date.getMonth()];
  const day = date.getDate();
  const year = date.getFullYear();

  // Format the date string
  const formattedDate = `${month} ${day} ${year}`;

  return formattedDate;
}
