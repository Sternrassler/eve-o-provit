"use client";

import React from "react";
import { RefreshCw } from "lucide-react";
import { useTradingSkills } from "@/lib/trading-skills-context";
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardAction } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

interface SkillBarProps {
  name: string;
  level: number;
  description?: string;
}

function SkillBar({ name, level, description }: SkillBarProps) {
  const maxLevel = 5;
  const percentage = (level / maxLevel) * 100;

  // Color based on skill level
  const getColor = (level: number) => {
    if (level === 0) return "bg-gray-300 dark:bg-gray-700";
    if (level <= 2) return "bg-yellow-500 dark:bg-yellow-600";
    if (level <= 4) return "bg-blue-500 dark:bg-blue-600";
    return "bg-green-500 dark:bg-green-600";
  };

  return (
    <div className="space-y-1">
      <div className="flex justify-between items-center text-sm">
        <span className="font-medium">{name}</span>
        <span className="text-muted-foreground">
          Level {level} / {maxLevel}
        </span>
      </div>
      {description && (
        <p className="text-xs text-muted-foreground">{description}</p>
      )}
      <div className="w-full h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
        <div
          className={`h-full ${getColor(level)} transition-all duration-300`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

interface StandingBarProps {
  name: string;
  value: number;
  description?: string;
}

function StandingBar({ name, value, description }: StandingBarProps) {
  // Standing range: -10.0 to +10.0
  const percentage = ((value + 10) / 20) * 100; // Map -10..+10 to 0..100%

  // Color based on standing value
  const getColor = (standing: number) => {
    if (standing < 0) return "bg-red-500 dark:bg-red-600";
    if (standing === 0) return "bg-gray-400 dark:bg-gray-600";
    if (standing < 5) return "bg-yellow-500 dark:bg-yellow-600";
    return "bg-green-500 dark:bg-green-600";
  };

  return (
    <div className="space-y-1">
      <div className="flex justify-between items-center text-sm">
        <span className="font-medium">{name}</span>
        <span className="text-muted-foreground">
          {value.toFixed(2)} / 10.0
        </span>
      </div>
      {description && (
        <p className="text-xs text-muted-foreground">{description}</p>
      )}
      <div className="w-full h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
        <div
          className={`h-full ${getColor(value)} transition-all duration-300`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

interface SkillsSectionProps {
  title: string;
  skills: Array<{
    name: string;
    level: number;
    description?: string;
  }>;
}

function SkillsSection({ title, skills }: SkillsSectionProps) {
  return (
    <div className="space-y-3">
      <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {title}
      </h4>
      <div className="space-y-3">
        {skills.map((skill) => (
          <SkillBar
            key={skill.name}
            name={skill.name}
            level={skill.level}
            description={skill.description}
          />
        ))}
      </div>
    </div>
  );
}

export function SkillsDisplay() {
  const { skills, loading, error, refreshSkills } = useTradingSkills();
  const [isRefreshing, setIsRefreshing] = React.useState(false);

  const handleRefresh = async () => {
    setIsRefreshing(true);
    await refreshSkills();
    setIsRefreshing(false);
  };

  if (loading && !skills) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Character Skills</CardTitle>
          <CardDescription>Loading your trading skills...</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (!skills) {
    return null;
  }

  const tradingSkills = [
    {
      name: "Accounting",
      level: skills.Accounting,
      description: `Sales Tax reduction: ${skills.Accounting * 10}%`,
    },
    {
      name: "Broker Relations",
      level: skills.BrokerRelations,
      description: `Broker Fee reduction: ${skills.BrokerRelations * 0.3}%`,
    },
    {
      name: "Advanced Broker Relations",
      level: skills.AdvancedBrokerRelations,
      description: `Additional Broker Fee reduction: ${skills.AdvancedBrokerRelations * 0.3}%`,
    },
  ];

  const standingsData = [
    {
      name: "Faction Standing",
      value: skills.FactionStanding,
      description: `Broker Fee reduction: ${(skills.FactionStanding * 0.03).toFixed(2)}%`,
    },
    {
      name: "Corp Standing",
      value: skills.CorpStanding,
      description: `Broker Fee reduction: ${(skills.CorpStanding * 0.02).toFixed(2)}%`,
    },
  ];

  // Calculate total broker fee reduction and effective fee
  const skillsReduction = skills.BrokerRelations * 0.3 + skills.AdvancedBrokerRelations * 0.3;
  const standingsReduction = skills.FactionStanding * 0.03 + skills.CorpStanding * 0.02;
  const totalBrokerReduction = skillsReduction + standingsReduction;
  const effectiveBrokerFee = Math.max(1.0, 3.0 - totalBrokerReduction);

  const navigationSkills = [
    {
      name: "Navigation",
      level: skills.Navigation,
      description: `Ship velocity: +${skills.Navigation * 5}%`,
    },
    {
      name: "Evasive Maneuvering",
      level: skills.EvasiveManeuvering,
      description: `Agility: +${skills.EvasiveManeuvering * 5}%`,
    },
  ];

  const cargoSkills = [
    {
      name: "Spaceship Command",
      level: skills.SpaceshipCommand,
      description: `Cargo capacity: +${skills.SpaceshipCommand * 5}%`,
    },
    {
      name: "Gallente Industrial",
      level: skills.GallenteIndustrial,
      description: `Cargo (Gallente): +${skills.GallenteIndustrial * 5}%`,
    },
    {
      name: "Caldari Industrial",
      level: skills.CaldariIndustrial,
      description: `Cargo (Caldari): +${skills.CaldariIndustrial * 5}%`,
    },
    {
      name: "Amarr Industrial",
      level: skills.AmarrIndustrial,
      description: `Cargo (Amarr): +${skills.AmarrIndustrial * 5}%`,
    },
    {
      name: "Minmatar Industrial",
      level: skills.MinmatarIndustrial,
      description: `Cargo (Minmatar): +${skills.MinmatarIndustrial * 5}%`,
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Character Skills</CardTitle>
        <CardDescription>
          {error ? (
            <span className="text-destructive">Failed to fetch skills - using defaults</span>
          ) : (
            "Auto-fetched from ESI API (cached for 5 minutes)"
          )}
        </CardDescription>
        <CardAction>
          <Button
            size="icon-sm"
            variant="ghost"
            onClick={handleRefresh}
            disabled={isRefreshing}
            title="Refresh skills"
          >
            <RefreshCw className={isRefreshing ? "animate-spin" : ""} />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-6">
        <SkillsSection title="Trading Skills" skills={tradingSkills} />
        
        {/* Standings Section */}
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            NPC Standings
          </h4>
          <div className="space-y-3">
            {standingsData.map((standing) => (
              <StandingBar
                key={standing.name}
                name={standing.name}
                value={standing.value}
                description={standing.description}
              />
            ))}
          </div>
          <div className="pt-2 text-xs text-muted-foreground bg-muted/50 p-3 rounded-md">
            <div className="flex justify-between items-center">
              <span>Effective Broker Fee:</span>
              <span className="font-semibold text-foreground">{effectiveBrokerFee.toFixed(2)}%</span>
            </div>
            <p className="mt-1 text-xs">
              Base 3.0% - Skills {skillsReduction.toFixed(1)}% - Standings {standingsReduction.toFixed(2)}% = {effectiveBrokerFee.toFixed(2)}% (min 1.0%)
            </p>
          </div>
        </div>

        <SkillsSection title="Navigation Skills" skills={navigationSkills} />
        <SkillsSection title="Cargo Skills" skills={cargoSkills} />
      </CardContent>
    </Card>
  );
}
