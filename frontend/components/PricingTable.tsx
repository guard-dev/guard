'use client';

import React from 'react';
import { Button } from "@/components/ui/button";
import { useRouter } from 'next/navigation';

interface PricingTableProps {
  handleCheckout?: (lookupString: string) => any;
};

const PricingTable: React.FC<PricingTableProps> = ({ handleCheckout }) => {
  const plans = [
    {
      name: 'Open Source',
      price: 'Free',
      extraPrice: '-',
      resources: 'Unlimited Resources',
      support: 'Community Support',
      features: [],
      cta: 'Self Host',
      lookupKey: '',
      href: "https://www.github.com/guard-dev/guard"
    },
    {
      name: 'Basic',
      price: '$150/mo',
      extraPrice: '$6 per 100 extra resources',
      resources: '1,000 Resources Included',
      support: 'Standard Support',
      features: [],
      cta: 'Get Started',
      lookupKey: 'guard_basic_monthly',
      href: "/console"
    },
    {
      name: 'Pro',
      price: '$400/mo',
      extraPrice: '$5 per 100 extra resources',
      resources: '5,000 Resources Included',
      support: 'Priority Support',
      features: [],
      cta: 'Get Started',
      lookupKey: 'guard_pro_monthly',
      href: "/console"
    },
    {
      name: 'Enterprise',
      price: 'Contact Us',
      extraPrice: '',
      resources: 'Unlimited Resources',
      support: 'Dedicated Support',
      features: [],
      cta: 'Contact Us',
      lookupKey: '',
      href: "https://www.cal.com/guard"
    },
  ];

  const router = useRouter();

  return (
    <div className="w-full mx-auto font-[family-name:var(--font-geist-mono)]">
      <h2 className="text-3xl mb-12 text-center">Pricing Plans</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {plans.map((plan) => (
          <div
            key={plan.name}
            className={`flex flex-col p-6 rounded-lg border bg-gradient-to-br from-white via-neutral-100/80 to-neutral-100 dark:from-neutral-900 dark:via-neutral-900/50 dark:to-neutral-900/30 hover:shadow-lg transition-all duration-300 ${plan.name === 'Pro' ? 'bg-gradient-to-br from-neutral-100 via-white to-neutral-50 dark:from-neutral-800 dark:via-neutral-900 dark:to-neutral-950 ring-2 ring-neutral-300 dark:ring-white/20' : ''
              }`}
          >
            <div className="mb-8">
              <h3 className="text-2xl font-bold mb-2">{plan.name}</h3>
              <div className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-neutral-950/90 via-neutral-500 to-neutral-950/90 dark:from-white/90 dark:via-neutral-400 dark:to-white/90">
                {plan.price}
              </div>
              {plan.extraPrice !== '-' && plan.extraPrice !== '' && (
                <div className="text-sm text-foreground/60 mt-2">
                  Extra: {plan.extraPrice}
                </div>
              )}
            </div>

            <div className="flex-grow space-y-4 mb-8">
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 rounded-full bg-gradient-to-r from-neutral-800 to-neutral-600 dark:from-white dark:to-neutral-400" />
                <span>{plan.resources}</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-4 h-4 rounded-full bg-gradient-to-r from-neutral-800 to-neutral-600 dark:from-white dark:to-neutral-400" />
                <span>{plan.support}</span>
              </div>
            </div>

            <a
              href={plan.href}
              target={plan.href.startsWith('http') ? '_blank' : undefined}
              rel={plan.href.startsWith('http') ? 'noopener noreferrer' : undefined}
              className="mt-auto"
            >
              <Button
                className={`w-full rounded-full ${plan.name === 'Basic' || plan.name === 'Pro'
                  ? 'bg-gradient-to-r from-neutral-900 via-neutral-800 to-neutral-700 hover:from-neutral-800 hover:via-neutral-700 hover:to-neutral-600 text-white border-0 dark:from-white dark:via-neutral-200 dark:to-neutral-400 dark:hover:from-neutral-200 dark:hover:via-neutral-300 dark:hover:to-neutral-500 dark:text-neutral-900'
                  : 'bg-gradient-to-r from-neutral-100 via-white to-neutral-200 hover:from-white hover:via-neutral-100 hover:to-neutral-300 dark:from-neutral-800 dark:via-neutral-900 dark:to-neutral-950 dark:hover:from-neutral-700 dark:hover:via-neutral-800 dark:hover:to-neutral-900 border border-neutral-200 dark:border-neutral-700 text-neutral-900 dark:text-white'
                  }`}
                variant={plan.name === 'Basic' || plan.name === 'Pro' ? 'default' : 'outline'}
                onClick={() => {
                  if (handleCheckout && plan.lookupKey) {
                    handleCheckout(plan.lookupKey);
                  }
                }}
              >
                {plan.cta}
              </Button>
            </a>
          </div>
        ))}
      </div>
    </div>
  );
};

export default PricingTable;
