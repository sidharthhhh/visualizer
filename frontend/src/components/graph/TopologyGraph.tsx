import { useEffect, useRef, useCallback } from 'react';
import cytoscape from 'cytoscape';
import { Container, Network, Edge } from '../../lib/api';
import { useAppStore } from '../../lib/store';

type HeatmapMode = 'none' | 'cpu' | 'memory';

interface TopologyGraphProps {
  containers: Container[];
  networks: Network[];
  edges: Edge[];
  searchQuery: string;
  heatmapMode?: HeatmapMode;
}

export default function TopologyGraph({ containers, networks, edges, searchQuery, heatmapMode = 'none' }: TopologyGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const setSelectedContainer = useAppStore((s) => s.setSelectedContainer);

  const buildElements = useCallback(() => {
    const nodes: cytoscape.ElementDefinition[] = [];
    const edgeDefs: cytoscape.ElementDefinition[] = [];

    networks.forEach((net) => {
      nodes.push({
        data: {
          id: `network-${net.id}`,
          label: net.name,
          type: 'network',
          driver: net.driver,
          subnet: net.subnet,
        },
        classes: 'network',
      });
    });

    containers.forEach((ctr) => {
      const matchesSearch =
        !searchQuery ||
        ctr.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        ctr.image.toLowerCase().includes(searchQuery.toLowerCase());

      let heatmapClass = '';
      if (heatmapMode === 'cpu') {
        heatmapClass = 'heatmap-cpu';
      } else if (heatmapMode === 'memory') {
        heatmapClass = 'heatmap-memory';
      }

      nodes.push({
        data: {
          id: `container-${ctr.id}`,
          label: ctr.name,
          type: 'container',
          image: ctr.image,
          state: ctr.state,
          container: ctr,
        },
        classes: `container ${ctr.state} ${matchesSearch ? '' : 'dimmed'} ${heatmapClass}`,
      });
    });

    const externalNodes = new Set<string>();
    edges.forEach((edge) => {
      if (edge.dst_container_id) {
        edgeDefs.push({
          data: {
            id: `edge-${edge.id}`,
            source: `container-${edge.src_container_id}`,
            target: `container-${edge.dst_container_id}`,
            protocol: edge.protocol,
            dst_port: edge.dst_port,
          },
          classes: `edge ${edge.protocol}`,
        });
      } else {
        const externalId = `external-${edge.dst_ip}`;
        if (!externalNodes.has(externalId)) {
          externalNodes.add(externalId);
          nodes.push({
            data: {
              id: externalId,
              label: edge.dst_ip,
              type: 'external',
            },
            classes: 'external',
          });
        }
        edgeDefs.push({
          data: {
            id: `edge-${edge.id}`,
            source: `container-${edge.src_container_id}`,
            target: externalId,
            protocol: edge.protocol,
            dst_port: edge.dst_port,
          },
          classes: `edge ${edge.protocol}`,
        });
      }
    });

    return { nodes, edges: edgeDefs };
  }, [containers, networks, edges, searchQuery, heatmapMode]);

  useEffect(() => {
    if (!containerRef.current) return;

    const cy = cytoscape({
      container: containerRef.current,
      elements: buildElements().nodes.concat(buildElements().edges),
      style: [
        {
          selector: 'node',
          style: {
            label: 'data(label)',
            'text-valign': 'bottom',
            'text-halign': 'center',
            'font-size': '10px',
            color: '#e0e0e8',
            'text-outline-color': '#0a0a0f',
            'text-outline-width': 2,
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container',
          style: {
            shape: 'round-rectangle',
            width: 60,
            height: 40,
            'background-color': '#1a1a2e',
            'border-width': 2,
            'border-color': '#00d4ff',
            'text-valign': 'bottom',
            'text-margin-y': 5,
            'transition-property': 'background-color, border-color, opacity',
            'transition-duration': 0.3,
          } as unknown as cytoscape.Css.Node,
        },
        {
          selector: '.container.running',
          style: {
            'background-color': '#1a1a2e',
            'border-color': '#00d4ff',
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.stopped',
          style: {
            'background-color': '#1a1a1a',
            'border-color': '#444',
            opacity: 0.6,
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.paused',
          style: {
            'background-color': '#1a1a2e',
            'border-color': '#ffaa00',
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.restarting',
          style: {
            'background-color': '#1a1a2e',
            'border-color': '#ffaa00',
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.dead',
          style: {
            'background-color': '#1a1a1a',
            'border-color': '#ff4444',
            opacity: 0.4,
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.heatmap-cpu',
          style: {
            'background-color': '#ff4444',
            'border-color': '#ff0000',
            opacity: 0.8,
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.heatmap-memory',
          style: {
            'background-color': '#a855f7',
            'border-color': '#9333ea',
            opacity: 0.8,
          } as cytoscape.Css.Node,
        },
        {
          selector: '.container.dimmed',
          style: {
            opacity: 0.2,
          } as cytoscape.Css.Node,
        },
        {
          selector: '.external',
          style: {
            shape: 'ellipse',
            width: 30,
            height: 30,
            'background-color': '#2a2a3e',
            'border-width': 1,
            'border-color': '#444',
            'text-valign': 'bottom',
            'text-margin-y': 5,
            'font-size': '9px',
            color: '#8888a0',
          } as cytoscape.Css.Node,
        },
        {
          selector: '.network',
          style: {
            shape: 'round-rectangle',
            width: 150,
            height: 100,
            'background-color': '#12121a',
            'background-opacity': 0.5,
            'border-width': 1,
            'border-color': '#2a2a3e',
            'border-style': 'dashed',
            'text-valign': 'top',
            'text-margin-y': 10,
            'font-size': '11px',
            color: '#8888a0',
          } as cytoscape.Css.Node,
        },
        {
          selector: 'edge',
          style: {
            width: 2,
            'line-color': '#333',
            'target-arrow-color': '#333',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            'transition-property': 'line-color, width',
            'transition-duration': 0.3,
          } as unknown as cytoscape.Css.Edge,
        },
        {
          selector: 'edge.tcp',
          style: {
            'line-color': '#00d4ff',
            'target-arrow-color': '#00d4ff',
          } as cytoscape.Css.Edge,
        },
        {
          selector: 'edge.udp',
          style: {
            'line-color': '#a855f7',
            'target-arrow-color': '#a855f7',
            'line-style': 'dashed',
          } as cytoscape.Css.Edge,
        },
      ],
      layout: {
        name: 'cose',
        animate: true,
        animationDuration: 500,
        nodeRepulsion: () => 8000,
        idealEdgeLength: () => 100,
        padding: 50,
      } as cytoscape.LayoutOptions,
      minZoom: 0.3,
      maxZoom: 3,
    });

    cy.on('tap', 'node.container', (event) => {
      const node = event.target;
      const container = node.data('container');
      if (container) {
        setSelectedContainer(container);
        cy.elements().removeClass('highlighted');
        node.addClass('highlighted');
        node.connectedEdges().addClass('highlighted');
        node.neighborhood().addClass('highlighted');
      }
    });

    cy.on('tap', (event) => {
      if (event.target === cy) {
        setSelectedContainer(null);
        cy.elements().removeClass('highlighted');
      }
    });

    cyRef.current = cy;

    return () => {
      cy.destroy();
    };
  }, [buildElements, setSelectedContainer]);

  useEffect(() => {
    if (cyRef.current) {
      const { nodes, edges: newEdges } = buildElements();
      const cy = cyRef.current;
      
      const existingIds = new Set(cy.elements().map((ele) => ele.id()));
      const newIds = new Set(nodes.concat(newEdges).map((ele) => ele.data.id as string));

      cy.elements().forEach((ele) => {
        if (!newIds.has(ele.id())) {
          ele.animate({
            style: { opacity: 0 },
            duration: 300,
            complete: () => {
              ele.remove();
            },
          });
        }
      });

      const newElements = nodes.concat(newEdges).filter((ele) => !existingIds.has(ele.data.id as string));
      if (newElements.length > 0) {
        const addedEles = cy.add(newElements);
        addedEles.forEach((ele) => {
          ele.style('opacity', 0);
          ele.animate({
            style: { opacity: 1 },
            duration: 300,
          });
        });
      }

      nodes.forEach((node) => {
        const existing = cy.getElementById(node.data.id as string);
        if (existing.length > 0 && node.classes) {
          existing.removeClass('running stopped paused restarting dead dimmed heatmap-cpu heatmap-memory');
          existing.addClass(node.classes);
        }
      });

      cy.layout({ name: 'cose', animate: true, animationDuration: 300 } as cytoscape.LayoutOptions).run();
    }
  }, [buildElements]);

  return (
    <div
      ref={containerRef}
      style={{
        width: '100%',
        height: '100%',
        background: '#0a0a0f',
      }}
    />
  );
}
