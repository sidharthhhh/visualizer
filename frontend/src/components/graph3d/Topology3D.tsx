import { useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls, Stars, Grid } from '@react-three/drei';
import { Container, Edge } from '../../lib/api';
import ContainerNode from './ContainerNode';
import EdgeTube from './EdgeTube';
import PacketStream from './PacketStream';

interface TopologySceneProps {
  containers: Container[];
  edges: Edge[];
  onContainerClick?: (container: Container) => void;
}

function TopologyScene({ containers, edges, onContainerClick }: TopologySceneProps) {
  const positions = useMemo(() => {
    const posMap = new Map<string, [number, number, number]>();
    const count = containers.length;
    const radius = Math.max(3, count * 0.5);

    containers.forEach((ctr, i) => {
      const angle = (i / count) * Math.PI * 2;
      const x = Math.cos(angle) * radius;
      const z = Math.sin(angle) * radius;
      const y = (Math.random() - 0.5) * 2;
      posMap.set(ctr.id, [x, y, z]);
    });

    return posMap;
  }, [containers]);

  return (
    <>
      {containers.map((ctr) => {
        const pos = positions.get(ctr.id) || [0, 0, 0];
        return (
          <ContainerNode
            key={ctr.id}
            position={pos}
            name={ctr.name}
            state={ctr.state}
            onClick={() => onContainerClick?.(ctr)}
          />
        );
      })}

      {edges.map((edge) => {
        const startPos = positions.get(edge.src_container_id);
        const endPos = edge.dst_container_id
          ? positions.get(edge.dst_container_id)
          : null;

        if (!startPos) return null;

        const defaultEnd: [number, number, number] = [
          startPos[0] + 3,
          startPos[1],
          startPos[2] + 3,
        ];

        return (
          <group key={edge.id}>
            <EdgeTube
              start={startPos}
              end={endPos || defaultEnd}
              protocol={edge.protocol}
            />
            <PacketStream
              start={startPos}
              end={endPos || defaultEnd}
              protocol={edge.protocol}
              bandwidth={0}
              packetCount={2}
            />
          </group>
        );
      })}
    </>
  );
}

interface Topology3DProps {
  containers: Container[];
  edges: Edge[];
  onContainerClick?: (container: Container) => void;
}

export default function Topology3D({ containers, edges, onContainerClick }: Topology3DProps) {
  return (
    <Canvas
      camera={{ position: [0, 5, 10], fov: 60 }}
      style={{ background: '#0a0a0f' }}
    >
      <ambientLight intensity={0.3} />
      <pointLight position={[10, 10, 10]} intensity={0.5} />
      <pointLight position={[-10, -10, -10]} intensity={0.3} color="#a855f7" />

      <Stars
        radius={100}
        depth={50}
        count={1000}
        factor={4}
        saturation={0}
        fade
        speed={1}
      />

      <Grid
        position={[0, -2, 0]}
        args={[20, 20]}
        cellSize={1}
        cellThickness={0.5}
        cellColor="#1a1a2e"
        sectionSize={5}
        sectionThickness={1}
        sectionColor="#2a2a3e"
        fadeDistance={20}
        fadeStrength={1}
        followCamera={false}
        infiniteGrid
      />

      <TopologyScene
        containers={containers}
        edges={edges}
        onContainerClick={onContainerClick}
      />

      <OrbitControls
        enablePan={true}
        enableZoom={true}
        enableRotate={true}
        minDistance={3}
        maxDistance={50}
        autoRotate
        autoRotateSpeed={0.5}
      />
    </Canvas>
  );
}
