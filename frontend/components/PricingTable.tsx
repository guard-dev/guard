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
    <div className="w-full mx-auto mt-8 font-[family-name:var(--font-geist-mono)]">
      <h2 className="text-3xl font-medium text-left mb-6">Pricing Plans</h2>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse">
          <thead>
            <tr className="bg-neutral-100 dark:bg-neutral-900">
              <th className="p-4 text-left border"></th>
              {plans.map((plan) => (
                <th key={plan.name} className="p-4 text-left border">
                  <div className="text-2xl font-bold mb-2">{plan.name}</div>
                  <div className="text-3xl font-bold text-blue-600">{plan.price}</div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="p-4 border font-bold">Resources</td>
              {plans.map((plan) => (
                <td key={plan.name} className="p-4 border text-left">
                  <div>{plan.resources}</div>
                  {plan.extraPrice !== '-' && plan.extraPrice !== '' && (
                    <div className="text-sm text-neutral-600 dark:text-neutral-400 mt-2">
                      Extra: {plan.extraPrice}
                    </div>
                  )}
                </td>
              ))}
            </tr>
            <tr>
              <td className="p-4 border font-bold">Support</td>
              {plans.map((plan) => (
                <td key={plan.name} className="p-4 border text-left">{plan.support}</td>
              ))}
            </tr>
            <tr>
              <td className="p-4 border"></td>
              {plans.map((plan) => (
                <td key={plan.name} className="p-4 border text-left">
                  <Button className="rounded-full" onClick={() => {
                    if (handleCheckout && plan.lookupKey.length) {
                      handleCheckout(plan.lookupKey)
                    } else {
                      router.push(plan.href)
                    }
                  }}>
                    {plan.cta}
                  </Button>
                </td>
              ))}
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default PricingTable;
