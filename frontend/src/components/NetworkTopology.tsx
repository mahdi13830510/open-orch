import React, { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import { Deployment, Repository } from '../types';

interface Node extends d3.SimulationNodeDatum {
  id: string;
  name: string;
  kind: string;
  state: string;
}

interface Link extends d3.SimulationLinkDatum<Node> {
  source: string;
  target: string;
  required?: boolean;
}

interface NetworkTopologyProps {
  deployments: Deployment[];
  repositories: Repository[];
  dependencies: Record<string, { target: string; required?: boolean }[]>;
}

export const NetworkTopology: React.FC<NetworkTopologyProps> = ({ 
  deployments, 
  repositories,
  dependencies 
}) => {
  const svgRef = useRef<SVGSVGElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!svgRef.current || !containerRef.current) return;

    let simulation: d3.Simulation<Node, undefined>;

    const init = () => {
      const svg = d3.select(svgRef.current);
      const width = containerRef.current?.clientWidth || 800;
      const height = containerRef.current?.clientHeight || 600;

      svg.attr('viewBox', [0, 0, width, height]);
      svg.selectAll('*').remove();

      // Definitions
      const defs = svg.append('defs');
      
      // Grid pattern
      const patternSize = 40;
      defs.append('pattern')
        .attr('id', 'grid')
        .attr('width', patternSize)
        .attr('height', patternSize)
        .attr('patternUnits', 'userSpaceOnUse')
        .append('path')
        .attr('d', `M ${patternSize} 0 L 0 0 0 ${patternSize}`)
        .attr('fill', 'none')
        .attr('stroke', 'rgba(20, 20, 20, 0.05)')
        .attr('stroke-width', '1');

      // Arrowhead marker
      defs.append('marker')
        .attr('id', 'arrowhead')
        .attr('viewBox', '0 -5 10 10')
        .attr('refX', 75) // Padding for node
        .attr('refY', 0)
        .attr('orient', 'auto')
        .attr('markerWidth', 5)
        .attr('markerHeight', 5)
        .append('path')
        .attr('d', 'M 0,-5 L 10 ,0 L 0,5')
        .attr('fill', '#141414');

      // Background
      svg.append('rect')
        .attr('width', '100%')
        .attr('height', '100%')
        .attr('fill', 'url(#grid)');

      const g = svg.append('g').attr('class', 'main-container');

      // Zoom
      const zoom = d3.zoom<SVGSVGElement, unknown>()
        .scaleExtent([0.05, 5])
        .on('zoom', (event) => {
          g.attr('transform', event.transform);
        });

      svg.call(zoom);

      // Data Processing
      const nodeMap = new Map<string, Node>();
      deployments.forEach(d => {
        if (!nodeMap.has(d.repository_id)) {
          const repo = repositories.find(r => r.id === d.repository_id);
          nodeMap.set(d.repository_id, {
            id: d.repository_id,
            name: repo?.name || d.container_name.split('-')[0],
            kind: repo?.service_kind || 'unknown',
            state: d.state,
          });
        }
      });

      const nodes = Array.from(nodeMap.values());
      const links: Link[] = [];
      const linkSet = new Set<string>();

      (Object.entries(dependencies) as [string, any[]][]).forEach(([sourceId, deps]) => {
        if (!nodeMap.has(sourceId)) return;
        deps.forEach(dep => {
          if (!nodeMap.has(dep.target) || sourceId === dep.target) return;
          const linkKey = `${sourceId}-${dep.target}`;
          if (!linkSet.has(linkKey)) {
            links.push({ source: sourceId, target: dep.target, required: dep.required });
            linkSet.add(linkKey);
          }
        });
      });

      // Simulation
      simulation = d3.forceSimulation<Node>(nodes)
        .force('link', d3.forceLink<Node, Link>(links).id(d => d.id).distance(240).strength(1))
        .force('charge', d3.forceManyBody().strength(-3500))
        .force('center', d3.forceCenter(width / 2, height / 2))
        .force('collision', d3.forceCollide().radius(140).strength(0.8));

      // Elements
      const link = g.append('g')
        .selectAll('path')
        .data(links)
        .join('path')
        .attr('fill', 'none')
        .attr('stroke', '#141414')
        .attr('stroke-width', d => d.required ? 2.5 : 1.5)
        .attr('stroke-opacity', 0.5)
        .attr('stroke-dasharray', d => d.required ? 'none' : '10,10')
        .attr('marker-end', 'url(#arrowhead)');

      const node = g.append('g')
        .selectAll('.node')
        .data(nodes)
        .join('g')
        .attr('class', 'node')
        .style('cursor', 'grab')
        .style('opacity', 0)
        .call(d3.drag<SVGGElement, Node>()
          .on('start', (event) => {
            if (!event.active) simulation.alphaTarget(0.3).restart();
            event.subject.fx = event.subject.x;
            event.subject.fy = event.subject.y;
          })
          .on('drag', (event) => {
            event.subject.fx = event.x;
            event.subject.fy = event.y;
          })
          .on('end', (event) => {
            if (!event.active) simulation.alphaTarget(0);
            event.subject.fx = null;
            event.subject.fy = null;
          }) as any);

      node.transition().duration(800).style('opacity', 1);

      const nodeWidth = 140;
      const nodeHeight = 60;

      // Card Shadow
      node.append('rect')
        .attr('width', nodeWidth)
        .attr('height', nodeHeight)
        .attr('x', -nodeWidth / 2 + 5)
        .attr('y', -nodeHeight / 2 + 5)
        .attr('fill', '#141414');

      // Card Main
      node.append('rect')
        .attr('width', nodeWidth)
        .attr('height', nodeHeight)
        .attr('x', -nodeWidth / 2)
        .attr('y', -nodeHeight / 2)
        .attr('fill', '#FFFFFF')
        .attr('stroke', '#141414')
        .attr('stroke-width', 2.5);

      // Status Indicator
      node.append('circle')
        .attr('r', 5)
        .attr('cx', -nodeWidth / 2 + 15)
        .attr('cy', -nodeHeight / 2 + 15)
        .attr('fill', d => {
          if (d.state === 'running' || d.state === 'healthy') return '#22C55E';
          if (d.state === 'failed') return '#EF4444';
          if (d.state === 'warning') return '#F59E0B';
          return '#94A3B8';
        })
        .attr('stroke', '#141414')
        .attr('stroke-width', 1.5);

      // Pulse animation for active nodes
      node.filter(d => d.state === 'running' || d.state === 'pending')
        .append('circle')
        .attr('r', 5)
        .attr('cx', -nodeWidth / 2 + 15)
        .attr('cy', -nodeHeight / 2 + 15)
        .attr('fill', 'none')
        .attr('stroke', d => d.state === 'running' ? '#22C55E' : '#F59E0B')
        .attr('stroke-width', 1)
        .append('animate')
          .attr('attributeName', 'r')
          .attr('from', '5')
          .attr('to', '12')
          .attr('dur', '1.5s')
          .attr('repeatCount', 'indefinite');

      node.filter(d => d.state === 'running' || d.state === 'pending')
        .selectAll('circle:last-child')
        .append('animate')
          .attr('attributeName', 'opacity')
          .attr('from', '0.8')
          .attr('to', '0')
          .attr('dur', '1.5s')
          .attr('repeatCount', 'indefinite');

      // Label
      node.append('text')
        .attr('y', 4)
        .attr('x', 5)
        .attr('text-anchor', 'middle')
        .attr('font-size', '10px')
        .attr('font-weight', '900')
        .attr('font-family', 'JetBrains Mono, monospace')
        .attr('fill', '#141414')
        .text(d => d.name.toUpperCase());

      // Icon & Kind
      node.append('text')
        .attr('x', -nodeWidth / 2 + 35)
        .attr('y', -nodeHeight / 2 + 20)
        .attr('font-size', '14px')
        .text(d => {
          const kind = d.kind.toLowerCase();
          if (kind === 'http' || d.name.includes('gateway') || d.name.includes('api')) return '🌐';
          if (kind === 'worker' || d.name.includes('processor')) return '⚙️';
          if (kind === 'database' || d.name.includes('db')) return '🗄️';
          if (kind === 'cache' || d.name.includes('redis')) return '⚡';
          return '📄';
        });

      node.append('text')
        .attr('x', nodeWidth / 2 - 10)
        .attr('y', nodeHeight / 2 - 10)
        .attr('font-size', '7px')
        .attr('font-weight', 'black')
        .attr('text-anchor', 'end')
        .attr('fill', '#141414')
        .attr('opacity', 0.4)
        .text(d => d.kind.toUpperCase());

      simulation.on('tick', () => {
        link.attr('d', d => {
          const s = d.source as any;
          const t = d.target as any;
          return `M ${s.x} ${s.y} L ${t.x} ${t.y}`;
        });
        node.attr('transform', d => `translate(${d.x},${d.y})`);
      });

      // Initial Zoom to fit with delay to allow simulation to settle slightly
      setTimeout(() => {
        if (nodes.length === 0) return;
        const bounds = g.node()?.getBBox();
        if (!bounds) return;
        
        const fullWidth = containerRef.current?.clientWidth || 800;
        const fullHeight = containerRef.current?.clientHeight || 600;
        
        const scale = 0.8 / Math.max(bounds.width / fullWidth, bounds.height / fullHeight);
        const midX = bounds.x + bounds.width / 2;
        const midY = bounds.y + bounds.height / 2;
        
        svg.transition().duration(1000).call(
          zoom.transform,
          d3.zoomIdentity
            .translate(fullWidth / 2, fullHeight / 2)
            .scale(Math.min(scale, 1.2))
            .translate(-midX, -midY)
        );
      }, 500);
    };

    init();

    const resizeObserver = new ResizeObserver(() => {
      if (simulation) simulation.stop();
      init();
    });

    resizeObserver.observe(containerRef.current);

    return () => {
      if (simulation) simulation.stop();
      resizeObserver.disconnect();
    };
  }, [deployments, repositories, dependencies]);


  return (
    <div ref={containerRef} className="w-full h-full bg-[#E4E3E0] border-2 border-[#141414] overflow-hidden relative">
      <svg ref={svgRef} className="w-full h-full" />
    </div>
  );
};

