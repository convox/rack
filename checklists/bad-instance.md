# Bad Instance

- list instances

  `convox instances`

- get docker ps output

  ```
   echo "Paste instance id:"
   read instance_id
   convox instances ssh $instance_id docker ps | tee $instance_id-docker-ps.txt
  ```

- get dmesg output

  ```
   echo "Paste instance id:"
   read instance_id
   convox instances ssh $instance_id dmesg | tee $instance_id-dmesg.txt
  ```

- get ecs-logs

  ```
   echo "Paste instance id:"
   read instance_id
   convox instances ssh $instance_id cat /var/log/ecs/ecs-* | tee $instance_id-ecs-agent-logs.txt
  ```

- upload to #retrospective room in slack
